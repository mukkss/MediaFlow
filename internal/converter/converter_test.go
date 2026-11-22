package converter

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mukkss/MediaFlow/internal/detector"
	"golang.org/x/image/bmp"
)

func TestStandardImageConverter(t *testing.T) {
	tmpDir := t.TempDir()
	conv := NewStandardImageConverter()
	opts := &Options{JPEGQuality: 90}
	ctx := context.Background()

	// 1. Create dummy PNG
	pngPath := filepath.Join(tmpDir, "test.png")
	f, _ := os.Create(pngPath)
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	_ = png.Encode(f, img)
	f.Close()

	info, err := detector.Detect(pngPath)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	destJpg := filepath.Join(tmpDir, "test.jpg")
	if err := conv.Convert(ctx, pngPath, destJpg, info, opts); err != nil {
		t.Fatalf("png to jpg conversion failed: %v", err)
	}

	fi, err := os.Stat(destJpg)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("expected valid converted jpg file, got size %d", fi.Size())
	}

	// 2. Create dummy BMP
	bmpPath := filepath.Join(tmpDir, "test.bmp")
	fBmp, _ := os.Create(bmpPath)
	_ = bmp.Encode(fBmp, img)
	fBmp.Close()

	infoBmp, _ := detector.Detect(bmpPath)
	destBmpJpg := filepath.Join(tmpDir, "test_bmp.jpg")
	if err := conv.Convert(ctx, bmpPath, destBmpJpg, infoBmp, opts); err != nil {
		t.Fatalf("bmp to jpg conversion failed: %v", err)
	}

	fiBmp, err := os.Stat(destBmpJpg)
	if err != nil || fiBmp.Size() == 0 {
		t.Fatalf("expected valid converted jpg from bmp, got size %d", fiBmp.Size())
	}
}

func TestVideoConverter(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed, skipping video conversion test")
	}

	tmpDir := t.TempDir()
	vConv := NewVideoConverter()
	opts := &Options{
		VideoCRF:     23,
		VideoPreset:  "ultrafast",
		AudioBitrate: "128k",
	}

	// Generate a 1-second dummy video using ffmpeg lavfi test source
	srcVideo := filepath.Join(tmpDir, "sample.avi")
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=30", "-f", "lavfi", "-i", "sine=frequency=1000:duration=1", "-c:v", "mjpeg", "-c:a", "pcm_s16le", srcVideo)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to generate test video: %v", err)
	}

	info, err := detector.Detect(srcVideo)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	destMp4 := filepath.Join(tmpDir, "sample.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := vConv.Convert(ctx, srcVideo, destMp4, info, opts); err != nil {
		t.Fatalf("video conversion failed: %v", err)
	}

	fi, err := os.Stat(destMp4)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("expected valid converted mp4, got size %d", fi.Size())
	}
}
