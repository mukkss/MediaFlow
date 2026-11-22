package tests

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mukkss/MediaFlow/internal/converter"
	"github.com/mukkss/MediaFlow/internal/detector"
	"github.com/mukkss/MediaFlow/internal/output"
	"github.com/mukkss/MediaFlow/internal/queue"
	"github.com/mukkss/MediaFlow/internal/watcher"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

// TestComprehensiveFormatMatrix tests conversion for every supported image and video format.
func TestComprehensiveFormatMatrix(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "Camera_Import")
	outputDir := filepath.Join(inputDir, "Converted")
	_ = os.MkdirAll(inputDir, 0755)

	outMgr := output.NewManager(inputDir, outputDir)
	opts := &converter.Options{
		JPEGQuality:  90,
		VideoCRF:     23,
		VideoPreset:  "ultrafast",
		AudioBitrate: "128k",
	}

	hasFFmpeg := false
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		hasFFmpeg = true
	}

	// 1. Standard Images (PNG, BMP, TIFF)
	img := image.NewRGBA(image.Rect(0, 0, 80, 80))
	for x := 0; x < 80; x++ {
		for y := 0; y < 80; y++ {
			img.Set(x, y, color.RGBA{R: 50, G: 150, B: 200, A: 255})
		}
	}

	// PNG
	pngPath := filepath.Join(inputDir, "test.png")
	fPng, _ := os.Create(pngPath)
	_ = png.Encode(fPng, img)
	fPng.Close()

	// BMP
	bmpPath := filepath.Join(inputDir, "test.bmp")
	fBmp, _ := os.Create(bmpPath)
	_ = bmp.Encode(fBmp, img)
	fBmp.Close()

	// TIFF
	tiffPath := filepath.Join(inputDir, "test.tif")
	fTiff, _ := os.Create(tiffPath)
	_ = tiff.Encode(fTiff, img, &tiff.Options{})
	fTiff.Close()

	// 2. Camera RAW with embedded full-size JPEG (CR2, ARW, NEF)
	var rawBuffer bytes.Buffer
	rawBuffer.Write([]byte{0x49, 0x49, 0x2A, 0x00, 0x10, 0x00, 0x00, 0x00, 0x43, 0x52, 0x02, 0x00})
	rawBuffer.Write(make([]byte, 512))
	var jBuf bytes.Buffer
	_ = jpeg.Encode(&jBuf, img, &jpeg.Options{Quality: 92})
	rawBuffer.Write(jBuf.Bytes())

	cr2Path := filepath.Join(inputDir, "sample.cr2")
	_ = os.WriteFile(cr2Path, rawBuffer.Bytes(), 0644)

	nefPath := filepath.Join(inputDir, "sample.nef")
	_ = os.WriteFile(nefPath, rawBuffer.Bytes(), 0644)

	arwPath := filepath.Join(inputDir, "sample.arw")
	_ = os.WriteFile(arwPath, rawBuffer.Bytes(), 0644)

	// 3. Videos (AVI, MOV, MKV) if FFmpeg is available
	var videoFiles []string
	if hasFFmpeg {
		for _, ext := range []string{".avi", ".mov", ".mkv"} {
			vPath := filepath.Join(inputDir, "clip"+ext)
			cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc=duration=1:size=160x120:rate=15", "-c:v", "mjpeg", vPath)
			if err := cmd.Run(); err == nil {
				videoFiles = append(videoFiles, vPath)
			}
		}
	}

	// 4. Setup Queue and Run Conversion
	var mu sync.Mutex
	completed := make(map[string]bool)

	q := queue.NewQueue(4, outMgr, opts, func(job *queue.Job) {
		mu.Lock()
		defer mu.Unlock()
		if job.Status == queue.StatusCompleted {
			completed[job.SourcePath] = true
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)
	defer q.Stop()

	// Enqueue images
	filesToTest := []string{pngPath, bmpPath, tiffPath, cr2Path, nefPath, arwPath}
	filesToTest = append(filesToTest, videoFiles...)

	for _, f := range filesToTest {
		q.Enqueue(ctx, f)
	}

	// Wait up to 10 seconds for completion
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(completed)
		mu.Unlock()
		if count >= len(filesToTest) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	mu.Lock()
	for _, f := range filesToTest {
		if !completed[f] {
			t.Errorf("file was not completed: %s", filepath.Base(f))
		}
	}
	mu.Unlock()

	// Verify all converted files exist in Converted/
	for _, f := range filesToTest {
		ext := filepath.Ext(f)
		isImage := ext != ".avi" && ext != ".mov" && ext != ".mkv"
		dest := outMgr.ResolveTargetPath(f, isImage, false)
		if fi, err := os.Stat(dest); err != nil || fi.Size() == 0 {
			t.Errorf("converted output missing or empty for %s: %s", filepath.Base(f), dest)
		}
	}
}

// TestHighConcurrencyImports tests queue behavior under burst imports (50 simultaneous files).
func TestHighConcurrencyImports(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "Burst_Import")
	outputDir := filepath.Join(inputDir, "Converted")
	_ = os.MkdirAll(inputDir, 0755)

	outMgr := output.NewManager(inputDir, outputDir)
	opts := &converter.Options{JPEGQuality: 85}

	var mu sync.Mutex
	processedCount := 0

	q := queue.NewQueue(4, outMgr, opts, func(job *queue.Job) {
		if job.Status == queue.StatusCompleted || job.Status == queue.StatusSkipped {
			mu.Lock()
			processedCount++
			mu.Unlock()
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)
	defer q.Stop()

	// Generate 50 dummy PNG files
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for i := 1; i <= 50; i++ {
		p := filepath.Join(inputDir, fmt.Sprintf("burst_%03d.png", i))
		f, _ := os.Create(p)
		_ = png.Encode(f, img)
		f.Close()
		q.Enqueue(ctx, p)
	}

	// Wait up to 10 seconds
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := processedCount
		mu.Unlock()
		if count >= 50 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	mu.Lock()
	finalCount := processedCount
	mu.Unlock()

	if finalCount != 50 {
		t.Errorf("expected 50 burst files processed, got %d", finalCount)
	}
}

// TestFileStabilitySimulation tests that files currently being written/transferred are held.
func TestFileStabilitySimulation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "growing_file.dat")

	// Initially create an empty (0 byte) file
	_ = os.WriteFile(filePath, []byte{}, 0644)

	checker := watcher.NewStabilityChecker(100, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Stability checker should return false for 0-byte file
	isStable := checker.WaitUntilStable(ctx, filePath)
	if isStable {
		t.Errorf("expected 0-byte file to NOT be considered stable")
	}

	// Now write content and wait
	_ = os.WriteFile(filePath, []byte("completed file data"), 0644)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	isStable2 := checker.WaitUntilStable(ctx2, filePath)
	if !isStable2 {
		t.Errorf("expected completed file to become stable")
	}
}

// TestMP4CompatibilityInspection tests skipping compatible H.264/AAC MP4s and re-encoding others.
func TestMP4CompatibilityInspection(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil || exec.Command("ffprobe", "-version").Run() != nil {
		t.Skip("ffmpeg/ffprobe not available, skipping test")
	}

	tmpDir := t.TempDir()

	// Generate compatible MP4 (H.264 + AAC + faststart)
	compatiblePath := filepath.Join(tmpDir, "compatible.mp4")
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc=duration=1:size=160x120:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=1",
		"-c:v", "libx264", "-c:a", "aac", "-movflags", "+faststart", compatiblePath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to generate compatible test MP4: %v", err)
	}

	info, err := detector.Detect(compatiblePath)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	ffprobeBin := detector.FindFFprobeBinary("")
	if err := detector.InspectVideo(context.Background(), ffprobeBin, info); err != nil {
		t.Fatalf("inspect video failed: %v", err)
	}

	if !info.IsCompatible {
		t.Errorf("expected H.264+AAC MP4 to be marked as compatible, got IsCompatible=false")
	}
	if info.RequiresAction {
		t.Errorf("expected compatible MP4 to NOT require re-encoding")
	}
}
