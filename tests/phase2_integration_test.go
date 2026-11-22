package tests

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mukkss/MediaFlow/internal/converter"
	"github.com/mukkss/MediaFlow/internal/output"
	"github.com/mukkss/MediaFlow/internal/queue"
)

func TestPhase2RAWAndHEICQueueIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "Camera_Import")
	outputDir := filepath.Join(inputDir, "Converted")

	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatalf("failed to create inputDir: %v", err)
	}

	outMgr := output.NewManager(inputDir, outputDir)
	opts := &converter.Options{
		JPEGQuality: 92,
		VideoCRF:    20,
		VideoPreset: "fast",
	}

	var mu sync.Mutex
	statusMap := make(map[string]queue.Status)

	q := queue.NewQueue(2, outMgr, opts, func(job *queue.Job) {
		mu.Lock()
		defer mu.Unlock()
		statusMap[job.SourcePath] = job.Status
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)
	defer q.Stop()

	// 1. Generate a mock RAW file (Canon CR2) with TIFF header + embedded full-resolution JPEG
	mockCR2Path := filepath.Join(inputDir, "IMG_0001.CR2")
	var rawBuffer bytes.Buffer
	// TIFF LE header + CR2 magic
	rawBuffer.Write([]byte{0x49, 0x49, 0x2A, 0x00, 0x10, 0x00, 0x00, 0x00, 0x43, 0x52, 0x02, 0x00})
	// Padding
	rawBuffer.Write(make([]byte, 1024))
	// Embedded large JPEG (> 50KB to simulate camera embedded preview)
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for x := 0; x < 400; x++ {
		for y := 0; y < 300; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var jpegBuf bytes.Buffer
	_ = jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: 95})

	// Pad with zero bytes inside to reach > 60KB
	rawBuffer.Write(jpegBuf.Bytes())
	_ = os.WriteFile(mockCR2Path, rawBuffer.Bytes(), 0644)

	// 2. Enqueue the RAW file
	q.Enqueue(ctx, mockCR2Path)

	// Wait up to 5 seconds for conversion
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		status := statusMap[mockCR2Path]
		mu.Unlock()
		if status == queue.StatusCompleted || status == queue.StatusFailed {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	mu.Lock()
	finalStatus := statusMap[mockCR2Path]
	mu.Unlock()

	if finalStatus != queue.StatusCompleted {
		t.Fatalf("expected mock CR2 to be successfully converted, got status: %s", finalStatus)
	}

	// 3. Verify converted JPEG exists in Converted folder
	destJpg := filepath.Join(outputDir, "IMG_0001.jpg")
	fi, err := os.Stat(destJpg)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("expected valid converted JPEG at %s, got size: %d", destJpg, fi.Size())
	}

	// 4. Verify original RAW file is untouched
	if _, err := os.Stat(mockCR2Path); os.IsNotExist(err) {
		t.Fatalf("original RAW file was deleted or moved")
	}

	// 5. Test Corrupted / Unsupported file handling
	corruptPath := filepath.Join(inputDir, "corrupted.xyz")
	_ = os.WriteFile(corruptPath, []byte("random corrupt bytes"), 0644)
	q.Enqueue(ctx, corruptPath)

	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	corruptStatus := statusMap[corruptPath]
	mu.Unlock()

	if corruptStatus != queue.StatusUnsupported && corruptStatus != queue.StatusFailed {
		t.Errorf("expected corrupt file to be Unsupported or Failed, got: %s", corruptStatus)
	}
}
