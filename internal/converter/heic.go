package converter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mukkss/MediaFlow/internal/detector"
)

// HEICConverter decodes HEIC and camera 10-bit HIF files using libheif (or FFmpeg fallback).
type HEICConverter struct{}

// NewHEICConverter creates a new HEIC converter.
func NewHEICConverter() *HEICConverter {
	return &HEICConverter{}
}

func (h *HEICConverter) CanConvert(info *detector.MediaInfo) bool {
	return info.Category == detector.CategoryHEIC
}

// FindHEIFConvertBinary searches for heif-convert in bundled directory or PATH.
func FindHEIFConvertBinary() string {
	if execPath, err := os.Executable(); err == nil {
		appDir := filepath.Dir(execPath)
		candidates := []string{
			filepath.Join(appDir, "codecs", "heif-convert.exe"),
			filepath.Join(appDir, "heif-convert.exe"),
			filepath.Join(appDir, "codecs", "heif-convert"),
			filepath.Join(appDir, "heif-convert"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	if path, err := exec.LookPath("heif-convert"); err == nil {
		return path
	}

	return ""
}

// Convert converts HEIC/HIF to JPEG.
func (h *HEICConverter) Convert(ctx context.Context, sourcePath, destPath string, info *detector.MediaInfo, opts *Options) error {
	quality := opts.JPEGQuality
	if quality <= 0 || quality > 100 {
		quality = 92
	}

	heifBin := FindHEIFConvertBinary()
	if heifBin != "" {
		// Use libheif heif-convert: heif-convert -q <quality> <input> <output>
		cmd := exec.CommandContext(ctx, heifBin, "-q", strconv.Itoa(quality), sourcePath, destPath)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err == nil {
			if fi, err := os.Stat(destPath); err == nil && fi.Size() > 0 {
				return nil
			}
		}
	}

	// Fallback to FFmpeg
	ffmpegBin := FindFFmpegBinary(opts.FFmpegPath)
	// Map JPEG quality (1-100) to ffmpeg -q:v (1-31, where 1 is best)
	qscale := 31 - (quality * 30 / 100)
	if qscale < 2 {
		qscale = 2
	}

	cmd := exec.CommandContext(ctx, ffmpegBin,
		"-y",
		"-i", sourcePath,
		"-q:v", strconv.Itoa(qscale),
		destPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if len(errMsg) > 300 {
			errMsg = errMsg[len(errMsg)-300:]
		}
		return fmt.Errorf("failed to convert HEIC/HIF %s: %w (%s)", filepath.Base(sourcePath), err, errMsg)
	}

	fi, err := os.Stat(destPath)
	if err != nil || fi.Size() == 0 {
		return fmt.Errorf("HEIC conversion resulted in missing or empty output for %s", filepath.Base(sourcePath))
	}

	return nil
}
