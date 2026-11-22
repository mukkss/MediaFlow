package converter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mukkss/MediaFlow/internal/detector"
)

func TestHEICConverterCanConvert(t *testing.T) {
	c := NewHEICConverter()

	infoHEIC := &detector.MediaInfo{
		Format:   detector.FormatHEIC,
		Category: detector.CategoryHEIC,
	}
	if !c.CanConvert(infoHEIC) {
		t.Errorf("expected HEICConverter to handle CategoryHEIC")
	}

	infoVideo := &detector.MediaInfo{
		Format:   detector.FormatMOV,
		Category: detector.CategoryVideo,
	}
	if c.CanConvert(infoVideo) {
		t.Errorf("expected HEICConverter to reject CategoryVideo")
	}
}

func TestRAWConverterCanConvert(t *testing.T) {
	c := NewRAWConverter()

	rawFormats := []detector.Format{
		detector.FormatCanonCR2,
		detector.FormatCanonCR3,
		detector.FormatNikonNEF,
		detector.FormatSonyARW,
		detector.FormatFujiRAF,
		detector.FormatAdobeDNG,
	}

	for _, fmt := range rawFormats {
		info := &detector.MediaInfo{
			Format:   fmt,
			Category: detector.CategoryRAW,
		}
		if !c.CanConvert(info) {
			t.Errorf("expected RAWConverter to handle format %s", fmt)
		}
	}

	infoJpg := &detector.MediaInfo{
		Format:   detector.FormatJPEG,
		Category: detector.CategoryImage,
	}
	if c.CanConvert(infoJpg) {
		t.Errorf("expected RAWConverter to reject standard JPEG image")
	}
}

func TestHEICConversionWithDummyFile(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewHEICConverter()
	opts := &Options{JPEGQuality: 92}

	// Create an invalid / dummy HEIC file
	heicPath := filepath.Join(tmpDir, "sample.heic")
	_ = os.WriteFile(heicPath, []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c'}, 0644)

	info := &detector.MediaInfo{
		Path:     heicPath,
		Format:   detector.FormatHEIC,
		Category: detector.CategoryHEIC,
	}

	destPath := filepath.Join(tmpDir, "output.jpg")
	ctx := context.Background()

	// An invalid truncated HEIC file must return a clean error, not panic
	err := c.Convert(ctx, heicPath, destPath, info, opts)
	if err == nil {
		t.Errorf("expected conversion of dummy/corrupt HEIC to return an error, got nil")
	}
}

func TestRAWConversionWithDummyFile(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewRAWConverter()
	opts := &Options{JPEGQuality: 92}

	// Create an invalid / dummy CR3 file
	cr3Path := filepath.Join(tmpDir, "sample.cr3")
	_ = os.WriteFile(cr3Path, []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'c', 'r', 'x', ' '}, 0644)

	info := &detector.MediaInfo{
		Path:     cr3Path,
		Format:   detector.FormatCanonCR3,
		Category: detector.CategoryRAW,
	}

	destPath := filepath.Join(tmpDir, "output.jpg")
	ctx := context.Background()

	// Truncated RAW file must return a clean error, not crash
	err := c.Convert(ctx, cr3Path, destPath, info, opts)
	if err == nil {
		t.Errorf("expected conversion of corrupt RAW to return an error, got nil")
	}
}
