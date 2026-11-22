package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mukkss/MediaFlow/internal/converter"
	"github.com/mukkss/MediaFlow/internal/detector"
	"github.com/mukkss/MediaFlow/internal/logger"
	"github.com/mukkss/MediaFlow/internal/output"
)

// Status represents the state of a media job.
type Status string

const (
	StatusDiscovered  Status = "Discovered"
	StatusQueued      Status = "Queued"
	StatusProcessing  Status = "Processing"
	StatusCompleted   Status = "Completed"
	StatusSkipped     Status = "Skipped"
	StatusFailed      Status = "Failed"
	StatusUnsupported Status = "Unsupported"
)

// Job represents a single media conversion task.
type Job struct {
	ID         string
	SourcePath string
	Info       *detector.MediaInfo
	Status     Status
	DestPath   string
	Error      string
	StartTime  time.Time
	EndTime    time.Time
}

// Stats holds in-memory session statistics.
type Stats struct {
	TotalDetected    int64
	TotalConverted   int64
	TotalSkipped     int64
	TotalFailed      int64
	TotalUnsupported int64
	ActiveWorkers    int32
}

// Queue coordinates conversion workers, deduplication, and execution.
type Queue struct {
	workers       int
	jobsCh        chan *Job
	activeJobs    map[string]bool
	jobsMu        sync.Mutex
	wg            sync.WaitGroup
	outputManager *output.Manager
	options       *converter.Options
	videoConv     converter.MediaConverter
	imageConv     converter.MediaConverter
	stats         Stats
	onJobUpdate   func(job *Job)
}

// NewQueue creates a bounded processing queue.
func NewQueue(workers int, outMgr *output.Manager, opts *converter.Options, onJobUpdate func(job *Job)) *Queue {
	if workers < 1 {
		workers = 2
	}
	return &Queue{
		workers:       workers,
		jobsCh:        make(chan *Job, 5000),
		activeJobs:    make(map[string]bool),
		outputManager: outMgr,
		options:       opts,
		videoConv:     converter.NewVideoConverter(),
		imageConv:     converter.NewStandardImageConverter(),
		onJobUpdate:   onJobUpdate,
	}
}

// Start spawns worker goroutines to process incoming jobs.
func (q *Queue) Start(ctx context.Context) {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.workerLoop(ctx, i+1)
	}
}

// Stop closes the job channel and waits for active conversions to finish.
func (q *Queue) Stop() {
	close(q.jobsCh)
	q.wg.Wait()
}

// Enqueue submits a file path to the processing queue.
func (q *Queue) Enqueue(ctx context.Context, filePath string) {
	q.jobsMu.Lock()
	if q.activeJobs[filePath] {
		q.jobsMu.Unlock()
		return
	}
	q.activeJobs[filePath] = true
	q.jobsMu.Unlock()

	atomic.AddInt64(&q.stats.TotalDetected, 1)

	// 1. Detect media format
	info, err := detector.Detect(filePath)
	if err != nil {
		q.recordFailure(filePath, nil, fmt.Sprintf("Detection failed: %v", err))
		return
	}

	// 2. Unsupported format check
	if info.Category == detector.CategoryUnsupported {
		atomic.AddInt64(&q.stats.TotalUnsupported, 1)
		q.updateJob(&Job{
			SourcePath: filePath,
			Info:       info,
			Status:     StatusUnsupported,
			Error:      "Format not supported for auto-conversion",
		})
		q.releaseJob(filePath)
		return
	}

	// 3. If video is MP4, inspect with FFprobe for H.264 / AAC compatibility
	if info.Format == detector.FormatMP4 {
		ffprobeBin := detector.FindFFprobeBinary(q.options.FFprobePath)
		_ = detector.InspectVideo(ctx, ffprobeBin, info)
	}

	// 4. Check if file is already compatible or doesn't require action
	if !info.RequiresAction {
		atomic.AddInt64(&q.stats.TotalSkipped, 1)
		q.updateJob(&Job{
			SourcePath: filePath,
			Info:       info,
			Status:     StatusSkipped,
			Error:      info.SkipReason,
		})
		q.releaseJob(filePath)
		return
	}

	// 5. Filesystem duplicate prevention: check if target file already exists
	isImage := info.Category == detector.CategoryImage || info.Category == detector.CategoryRAW || info.Category == detector.CategoryHEIC
	if q.outputManager.DestinationExists(filePath, isImage) {
		atomic.AddInt64(&q.stats.TotalSkipped, 1)
		q.updateJob(&Job{
			SourcePath: filePath,
			Info:       info,
			Status:     StatusSkipped,
			Error:      "Destination file already exists in Converted folder",
		})
		q.releaseJob(filePath)
		return
	}

	// 6. Submit to workers channel
	job := &Job{
		SourcePath: filePath,
		Info:       info,
		Status:     StatusQueued,
		StartTime:  time.Now(),
	}
	q.updateJob(job)

	select {
	case q.jobsCh <- job:
	case <-ctx.Done():
		q.releaseJob(filePath)
	}
}

func (q *Queue) workerLoop(ctx context.Context, workerID int) {
	defer q.wg.Done()

	for job := range q.jobsCh {
		atomic.AddInt32(&q.stats.ActiveWorkers, 1)
		q.processJob(ctx, job)
		atomic.AddInt32(&q.stats.ActiveWorkers, -1)
	}
}

func (q *Queue) processJob(ctx context.Context, job *Job) {
	defer q.releaseJob(job.SourcePath)

	job.Status = StatusProcessing
	q.updateJob(job)

	isImage := job.Info.Category == detector.CategoryImage || job.Info.Category == detector.CategoryRAW || job.Info.Category == detector.CategoryHEIC
	finalDest := q.outputManager.ResolveTargetPath(job.SourcePath, isImage, true)
	tempDest := q.outputManager.TempPathFor(finalDest)
	job.DestPath = finalDest

	logger.Get().Info("Starting conversion", "source", job.SourcePath, "target", finalDest)

	var err error
	switch job.Info.Category {
	case detector.CategoryVideo:
		err = q.videoConv.Convert(ctx, job.SourcePath, tempDest, job.Info, q.options)
	case detector.CategoryImage:
		err = q.imageConv.Convert(ctx, job.SourcePath, tempDest, job.Info, q.options)
	default:
		err = fmt.Errorf("no converter registered for category %s", job.Info.Category)
	}

	if err != nil {
		atomic.AddInt64(&q.stats.TotalFailed, 1)
		job.Status = StatusFailed
		job.Error = err.Error()
		job.EndTime = time.Now()
		q.updateJob(job)
		logger.Get().Error("Conversion failed", "source", job.SourcePath, "err", err)
		return
	}

	// Finalize temp file atomically
	if err := q.outputManager.Finalize(tempDest, finalDest); err != nil {
		atomic.AddInt64(&q.stats.TotalFailed, 1)
		job.Status = StatusFailed
		job.Error = fmt.Sprintf("Finalization error: %v", err)
		job.EndTime = time.Now()
		q.updateJob(job)
		logger.Get().Error("Finalization failed", "temp", tempDest, "err", err)
		return
	}

	atomic.AddInt64(&q.stats.TotalConverted, 1)
	job.Status = StatusCompleted
	job.EndTime = time.Now()
	q.updateJob(job)
	logger.Get().Info("Successfully converted", "source", job.SourcePath, "dest", finalDest)
}

func (q *Queue) recordFailure(sourcePath string, info *detector.MediaInfo, errMsg string) {
	atomic.AddInt64(&q.stats.TotalFailed, 1)
	q.updateJob(&Job{
		SourcePath: sourcePath,
		Info:       info,
		Status:     StatusFailed,
		Error:      errMsg,
	})
	q.releaseJob(sourcePath)
}

func (q *Queue) releaseJob(path string) {
	q.jobsMu.Lock()
	delete(q.activeJobs, path)
	q.jobsMu.Unlock()
}

func (q *Queue) updateJob(job *Job) {
	if q.onJobUpdate != nil {
		q.onJobUpdate(job)
	}
}

// GetStats returns the current session statistics.
func (q *Queue) GetStats() Stats {
	return Stats{
		TotalDetected:    atomic.LoadInt64(&q.stats.TotalDetected),
		TotalConverted:   atomic.LoadInt64(&q.stats.TotalConverted),
		TotalSkipped:     atomic.LoadInt64(&q.stats.TotalSkipped),
		TotalFailed:      atomic.LoadInt64(&q.stats.TotalFailed),
		TotalUnsupported: atomic.LoadInt64(&q.stats.TotalUnsupported),
		ActiveWorkers:    atomic.LoadInt32(&q.stats.ActiveWorkers),
	}
}
