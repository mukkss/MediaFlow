package converter

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mukkss/MediaFlow/internal/detector"
)

// RAWConverter develops or extracts camera RAW formats (CR2, CR3, NEF, ARW, RAF, DNG) into JPEG.
type RAWConverter struct{}

// NewRAWConverter creates a camera RAW converter.
func NewRAWConverter() *RAWConverter {
	return &RAWConverter{}
}

func (r *RAWConverter) CanConvert(info *detector.MediaInfo) bool {
	return info.Category == detector.CategoryRAW
}

// FindDcrawEmuBinary searches for dcraw_emu in bundled codecs folder or PATH.
func FindDcrawEmuBinary() string {
	if execPath, err := os.Executable(); err == nil {
		appDir := filepath.Dir(execPath)
		candidates := []string{
			filepath.Join(appDir, "codecs", "dcraw_emu.exe"),
			filepath.Join(appDir, "dcraw_emu.exe"),
			filepath.Join(appDir, "codecs", "dcraw_emu"),
			filepath.Join(appDir, "dcraw_emu"),
			filepath.Join(appDir, "codecs", "dcraw.exe"),
			filepath.Join(appDir, "dcraw.exe"),
			filepath.Join(appDir, "dcraw"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	if path, err := exec.LookPath("dcraw_emu"); err == nil {
		return path
	}
	if path, err := exec.LookPath("dcraw"); err == nil {
		return path
	}

	return ""
}

// Convert converts camera RAW files to JPEG.
func (r *RAWConverter) Convert(ctx context.Context, sourcePath, destPath string, info *detector.MediaInfo, opts *Options) error {
	quality := opts.JPEGQuality
	if quality <= 0 || quality > 100 {
		quality = 92
	}

	// 1. Try dcraw_emu / dcraw if present
	dcrawBin := FindDcrawEmuBinary()
	if dcrawBin != "" {
		if err := r.convertWithDcraw(ctx, dcrawBin, sourcePath, destPath, quality); err == nil {
			return nil
		}
	}

	// 2. Try extracting embedded camera JPEG preview (fast, accurate camera profile)
	if err := r.extractEmbeddedJPEG(sourcePath, destPath, quality); err == nil {
		return nil
	}

	// 3. Fallback to FFmpeg
	ffmpegBin := FindFFmpegBinary(opts.FFmpegPath)
	cmd := exec.CommandContext(ctx, ffmpegBin, "-y", "-i", sourcePath, "-q:v", "2", destPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if len(errMsg) > 200 {
			errMsg = errMsg[len(errMsg)-200:]
		}
		return fmt.Errorf("failed to develop RAW image %s: %w (%s)", filepath.Base(sourcePath), err, errMsg)
	}

	fi, err := os.Stat(destPath)
	if err != nil || fi.Size() == 0 {
		return fmt.Errorf("RAW conversion produced no output for %s", filepath.Base(sourcePath))
	}

	return nil
}

func (r *RAWConverter) convertWithDcraw(ctx context.Context, bin, source, dest string, quality int) error {
	tempTiff := dest + ".raw.tiff"
	defer os.Remove(tempTiff)

	// dcraw_emu -w (camera white balance) -T (output TIFF) -o 1 (sRGB) -O <output> <source>
	cmd := exec.CommandContext(ctx, bin, "-w", "-T", "-o", "1", "-O", tempTiff, source)
	if err := cmd.Run(); err != nil {
		return err
	}

	f, err := os.Open(tempTiff)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
}

// extractEmbeddedJPEG scans the RAW file for the embedded full-resolution camera JPEG.
func (r *RAWConverter) extractEmbeddedJPEG(source, dest string, quality int) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil || len(data) < 1024 {
		return fmt.Errorf("insufficient data for RAW scan")
	}

	// Search for largest embedded JPEG chunk (SOI: FF D8 FF ... EOI: FF D9)
	var largestJPEG []byte
	offset := 0
	for {
		idx := bytes.Index(data[offset:], []byte{0xFF, 0xD8, 0xFF})
		if idx == -1 {
			break
		}
		start := offset + idx
		// Search for EOI
		endIdx := bytes.Index(data[start:], []byte{0xFF, 0xD9})
		if endIdx == -1 {
			offset = start + 3
			continue
		}
		end := start + endIdx + 2
		candidate := data[start:end]

		// Only consider embedded images of substantial size (> 2KB)
		if len(candidate) > 2048 && len(candidate) > len(largestJPEG) {
			// Verify candidate starts with valid SOI
			if bytes.HasPrefix(candidate, []byte{0xFF, 0xD8, 0xFF}) {
				largestJPEG = candidate
			}
		}
		offset = start + 3
	}

	if len(largestJPEG) == 0 {
		return fmt.Errorf("no embedded preview JPEG found in RAW file")
	}

	// Verify the extracted bytes are a valid JPEG image
	img, err := jpeg.Decode(bytes.NewReader(largestJPEG))
	if err != nil {
		return err
	}

	// Apply orientation if needed
	orientation := ReadOrientationTag(source)
	if orientation > 1 {
		img = ApplyOrientation(img, orientation)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
}
