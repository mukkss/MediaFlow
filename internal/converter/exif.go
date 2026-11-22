package converter

import (
	"bytes"
	"encoding/binary"
	"image"
	"io"
	"os"

	"golang.org/x/image/draw"
)

// ReadOrientationTag extracts the EXIF orientation tag from JPEG/TIFF if available.
func ReadOrientationTag(filePath string) int {
	f, err := os.Open(filePath)
	if err != nil {
		return 1
	}
	defer f.Close()

	buf := make([]byte, 65536)
	n, _ := io.ReadFull(f, buf)
	buf = buf[:n]

	// Find EXIF marker
	idx := bytes.Index(buf, []byte("Exif\x00\x00"))
	if idx == -1 {
		return 1
	}

	tiffData := buf[idx+6:]
	if len(tiffData) < 8 {
		return 1
	}

	var bo binary.ByteOrder
	if tiffData[0] == 'I' && tiffData[1] == 'I' {
		bo = binary.LittleEndian
	} else if tiffData[0] == 'M' && tiffData[1] == 'M' {
		bo = binary.BigEndian
	} else {
		return 1
	}

	firstIFD := bo.Uint32(tiffData[4:8])
	if int(firstIFD)+2 > len(tiffData) {
		return 1
	}

	entries := bo.Uint16(tiffData[firstIFD : firstIFD+2])
	offset := int(firstIFD) + 2

	for i := 0; i < int(entries); i++ {
		if offset+12 > len(tiffData) {
			break
		}
		tag := bo.Uint16(tiffData[offset : offset+2])
		if tag == 0x0112 { // Orientation Tag
			val := bo.Uint16(tiffData[offset+8 : offset+10])
			if val >= 1 && val <= 8 {
				return int(val)
			}
			return 1
		}
		offset += 12
	}

	return 1
}

// ApplyOrientation transforms an image.Image based on EXIF orientation flag.
func ApplyOrientation(src image.Image, orientation int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	switch orientation {
	case 2: // Flip Horizontal
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(w-1-x, y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	case 3: // Rotate 180
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(w-1-x, h-1-y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	case 4: // Flip Vertical
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(x, h-1-y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	case 6: // Rotate 90 CW
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(h-1-y, x, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	case 8: // Rotate 270 CW (90 CCW)
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(y, w-1-x, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	default:
		// Orientation 1 (Normal) or unsupported
		return src
	}
}

// EnsureRGBA converts an image to RGBA if not already
func EnsureRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		return rgba
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Copy(dst, image.Point{}, src, b, draw.Src, nil)
	return dst
}
