package converter

import (
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/mukkss/MediaFlow/internal/detector"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

// StandardImageConverter handles PNG, BMP, TIFF, and GIF to JPEG conversions.
type StandardImageConverter struct{}

// NewStandardImageConverter creates an image converter.
func NewStandardImageConverter() *StandardImageConverter {
	return &StandardImageConverter{}
}

func (s *StandardImageConverter) CanConvert(info *detector.MediaInfo) bool {
	return info.Category == detector.CategoryImage && info.Format != detector.FormatJPEG
}

// Convert decodes the source image and encodes it to JPEG at configured quality.
func (s *StandardImageConverter) Convert(ctx context.Context, sourcePath, destPath string, info *detector.MediaInfo, opts *Options) error {
	f, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source image: %w", err)
	}
	defer f.Close()

	var img image.Image

	switch info.Format {
	case detector.FormatPNG:
		img, err = png.Decode(f)
	case detector.FormatBMP:
		img, err = bmp.Decode(f)
	case detector.FormatTIFF:
		img, err = tiff.Decode(f)
	case detector.FormatGIF:
		// Decode first frame of GIF per spec
		img, err = gif.Decode(f)
	default:
		// Fallback to standard generic decoder
		img, _, err = image.Decode(f)
	}

	if err != nil {
		return fmt.Errorf("failed to decode image %s: %w", sourcePath, err)
	}

	// Apply EXIF orientation
	orientation := ReadOrientationTag(sourcePath)
	if orientation > 1 {
		img = ApplyOrientation(img, orientation)
	}

	// Create destination JPEG file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination JPEG %s: %w", destPath, err)
	}
	defer out.Close()

	quality := opts.JPEGQuality
	if quality <= 0 || quality > 100 {
		quality = 92
	}

	jpegOpts := &jpeg.Options{
		Quality: quality,
	}

	if err := jpeg.Encode(out, img, jpegOpts); err != nil {
		return fmt.Errorf("failed to encode JPEG %s: %w", destPath, err)
	}

	return nil
}
