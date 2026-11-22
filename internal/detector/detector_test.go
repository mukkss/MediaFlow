package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSignatures(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		filename     string
		header       []byte
		wantFormat   Format
		wantCategory Category
	}{
		{
			name:         "JPEG signature",
			filename:     "sample.jpg",
			header:       []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46},
			wantFormat:   FormatJPEG,
			wantCategory: CategoryImage,
		},
		{
			name:         "PNG signature",
			filename:     "sample.png",
			header:       []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			wantFormat:   FormatPNG,
			wantCategory: CategoryImage,
		},
		{
			name:         "BMP signature",
			filename:     "sample.bmp",
			header:       []byte{0x42, 0x4D, 0x00, 0x00, 0x00, 0x00},
			wantFormat:   FormatBMP,
			wantCategory: CategoryImage,
		},
		{
			name:         "GIF89a signature",
			filename:     "sample.gif",
			header:       []byte("GIF89a\x01\x00\x01\x00"),
			wantFormat:   FormatGIF,
			wantCategory: CategoryImage,
		},
		{
			name:         "AVI signature",
			filename:     "sample.avi",
			header:       []byte("RIFF\x00\x00\x00\x00AVI LIST"),
			wantFormat:   FormatAVI,
			wantCategory: CategoryVideo,
		},
		{
			name:         "Matroska MKV signature",
			filename:     "sample.mkv",
			header:       []byte{0x1A, 0x45, 0xDF, 0xA3, 0x93, 0x42, 0x82, 0x88},
			wantFormat:   FormatMKV,
			wantCategory: CategoryVideo,
		},
		{
			name:         "Canon CR3 signature",
			filename:     "sample.cr3",
			header:       []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'c', 'r', 'x', ' '},
			wantFormat:   FormatCanonCR3,
			wantCategory: CategoryRAW,
		},
		{
			name:         "Canon CR2 signature",
			filename:     "sample.cr2",
			header:       []byte{0x49, 0x49, 0x2A, 0x00, 0x10, 0x00, 0x00, 0x00, 0x43, 0x52, 0x02, 0x00},
			wantFormat:   FormatCanonCR2,
			wantCategory: CategoryRAW,
		},
		{
			name:         "Fujifilm RAF signature",
			filename:     "sample.raf",
			header:       []byte("FUJIFILMCCD-RAW 0201"),
			wantFormat:   FormatFujiRAF,
			wantCategory: CategoryRAW,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(filePath, tt.header, 0644); err != nil {
				t.Fatalf("failed to create dummy file: %v", err)
			}

			info, err := Detect(filePath)
			if err != nil {
				t.Fatalf("unexpected detection error: %v", err)
			}

			if info.Format != tt.wantFormat {
				t.Errorf("expected format %s, got %s", tt.wantFormat, info.Format)
			}
			if info.Category != tt.wantCategory {
				t.Errorf("expected category %s, got %s", tt.wantCategory, info.Category)
			}
		})
	}
}
