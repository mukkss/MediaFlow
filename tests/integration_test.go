package tests

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mukkss/MediaFlow/internal/converter"
	"github.com/mukkss/MediaFlow/internal/output"
	"github.com/mukkss/MediaFlow/internal/queue"
	"github.com/mukkss/MediaFlow/internal/watcher"
)

func TestFullPipelineIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "Camera_Import")
	outputDir := filepath.Join(inputDir, "Converted")

	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatalf("failed to create inputDir: %v", err)
	}

	outMgr := output.NewManager(inputDir, outputDir)
	opts := &converter.Options{
		JPEGQuality: 92,
		VideoCRF:    23,
		VideoPreset: "ultrafast",
	}

	var mu sync.Mutex
	completedFiles := make(map[string]bool)

	q := queue.NewQueue(2, outMgr, opts, func(job *queue.Job) {
		mu.Lock()
		defer mu.Unlock()
		if job.Status == queue.StatusCompleted {
			completedFiles[job.SourcePath] = true
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)
	defer q.Stop()

	// 1. Create a dummy image file in inputDir
	testImgPath := filepath.Join(inputDir, "photo1.png")
	f, _ := os.Create(testImgPath)
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for x := 0; x < 50; x++ {
		for y := 0; y < 50; y++ {
			img.Set(x, y, color.RGBA{G: 255, A: 255})
		}
	}
	_ = png.Encode(f, img)
	f.Close()

	// 2. Set up Scanner and Watcher
	onFile := func(path string) {
		q.Enqueue(ctx, path)
	}

	scanner := watcher.NewScanner(inputDir, outputDir, 1, onFile)
	scanner.ScanOnce()

	// Wait up to 5 seconds for queue processing
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := completedFiles[testImgPath]
		mu.Unlock()
		if done {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	mu.Lock()
	done := completedFiles[testImgPath]
	mu.Unlock()

	if !done {
		t.Fatalf("expected photo1.png to be completed by queue")
	}

	// 3. Verify output exists
	expectedDest := filepath.Join(outputDir, "photo1.jpg")
	if fi, err := os.Stat(expectedDest); err != nil || fi.Size() == 0 {
		t.Fatalf("expected converted JPEG at %s, got err: %v", expectedDest, err)
	}

	// 4. Verify original remains untouched
	if _, err := os.Stat(testImgPath); os.IsNotExist(err) {
		t.Fatalf("original file %s was deleted or moved", testImgPath)
	}

	// 5. Test duplicate prevention: scan again and ensure it is skipped
	scanner.ScanOnce()
	stats := q.GetStats()
	if stats.TotalSkipped < 1 {
		t.Errorf("expected duplicate file to be skipped, stats: %+v", stats)
	}
}
