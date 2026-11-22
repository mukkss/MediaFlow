package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/mukkss/MediaFlow/internal/config"
	"github.com/mukkss/MediaFlow/internal/converter"
	"github.com/mukkss/MediaFlow/internal/logger"
	"github.com/mukkss/MediaFlow/internal/output"
	"github.com/mukkss/MediaFlow/internal/queue"
	"github.com/mukkss/MediaFlow/internal/watcher"
)

// AppService manages the core conversion engine and bridges Go to Svelte frontend.
type AppService struct {
	cfg          *config.Config
	isMonitoring bool
	ctx          context.Context
	cancel       context.CancelFunc
	outMgr       *output.Manager
	q            *queue.Queue
	watcher      *watcher.Watcher
	scanner      *watcher.Scanner
	recentJobs   []*queue.Job
	mu           sync.RWMutex
	onEvent      func(event string, data any)
}

// NewAppService initializes the application bridge service.
func NewAppService(cfg *config.Config) *AppService {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &AppService{
		cfg:        cfg,
		recentJobs: make([]*queue.Job, 0, 50),
	}
}

// SetEventEmitter registers a callback for pushing real-time updates to Wails/Svelte.
func (s *AppService) SetEventEmitter(fn func(event string, data any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onEvent = fn
}

// IsMonitoring returns whether the folder watcher is active.
func (s *AppService) IsMonitoring() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isMonitoring
}

// Start activates folder monitoring and the conversion worker pool.
func (s *AppService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isMonitoring {
		return nil
	}

	if s.cfg.InputDir == "" {
		return fmt.Errorf("please select an input directory to monitor")
	}

	if _, err := os.Stat(s.cfg.InputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", s.cfg.InputDir)
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	effectiveOut := s.cfg.EffectiveOutputDir()
	s.outMgr = output.NewManager(s.cfg.InputDir, effectiveOut)
	s.outMgr.CleanStaleTemps()

	opts := &converter.Options{
		VideoCRF:     s.cfg.VideoCRF,
		VideoPreset:  s.cfg.VideoPreset,
		AudioBitrate: s.cfg.AudioBitrate,
		JPEGQuality:  s.cfg.JPEGQuality,
		FFmpegPath:   s.cfg.FFmpegPath,
		FFprobePath:  s.cfg.FFprobePath,
	}

	s.q = queue.NewQueue(s.cfg.WorkerCount, s.outMgr, opts, func(job *queue.Job) {
		s.recordJob(job)
		if s.onEvent != nil {
			s.onEvent("job_update", job)
			s.onEvent("stats_update", s.q.GetStats())
		}
	})
	s.q.Start(s.ctx)

	onFile := func(path string) {
		s.q.Enqueue(s.ctx, path)
	}

	var err error
	s.watcher, err = watcher.NewWatcher(s.cfg.InputDir, effectiveOut, s.cfg.StabilityIntervalMs, s.cfg.StabilityRequiredChecks, onFile)
	if err != nil {
		s.cancel()
		s.q.Stop()
		return fmt.Errorf("failed to start filesystem watcher: %w", err)
	}

	if err := s.watcher.Start(s.ctx); err != nil {
		s.cancel()
		s.q.Stop()
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	s.scanner = watcher.NewScanner(s.cfg.InputDir, effectiveOut, s.cfg.ScanIntervalSeconds, onFile)
	go s.scanner.Start(s.ctx)

	s.isMonitoring = true
	logger.Get().Info("AppService: Monitoring started", "input", s.cfg.InputDir, "output", effectiveOut)
	if s.onEvent != nil {
		s.onEvent("monitoring_state", true)
	}
	return nil
}

// Stop stops the watcher, scanner, and drains in-flight jobs.
func (s *AppService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isMonitoring {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	if s.q != nil {
		s.q.Stop()
	}

	s.isMonitoring = false
	logger.Get().Info("AppService: Monitoring stopped")
	if s.onEvent != nil {
		s.onEvent("monitoring_state", false)
	}
	return nil
}

// GetConfig returns the active configuration.
func (s *AppService) GetConfig() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// UpdateConfig applies and persists new configuration settings.
func (s *AppService) UpdateConfig(newCfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newCfg.Validate()
	s.cfg = newCfg
	_ = s.cfg.Save()

	logger.Get().Info("AppService: Config updated", "input", s.cfg.InputDir, "workers", s.cfg.WorkerCount)
	return nil
}

// GetStats returns current conversion counters.
func (s *AppService) GetStats() queue.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.q == nil {
		return queue.Stats{}
	}
	return s.q.GetStats()
}

// GetRecentJobs returns up to 50 recent jobs.
func (s *AppService) GetRecentJobs() []*queue.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*queue.Job, len(s.recentJobs))
	copy(out, s.recentJobs)
	return out
}

func (s *AppService) recordJob(job *queue.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update existing job in list if present
	for i, existing := range s.recentJobs {
		if existing.SourcePath == job.SourcePath {
			s.recentJobs[i] = job
			return
		}
	}

	// Insert newest at beginning
	if len(s.recentJobs) >= 50 {
		s.recentJobs = s.recentJobs[:49]
	}
	s.recentJobs = append([]*queue.Job{job}, s.recentJobs...)
}

// OpenFolder opens a folder in the native file explorer.
func (s *AppService) OpenFolder(path string) error {
	if path == "" {
		path = s.cfg.EffectiveOutputDir()
	}

	_ = os.MkdirAll(path, 0755)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", filepath.Clean(path))
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}
