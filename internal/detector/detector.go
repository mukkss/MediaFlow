package detector

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Detect inspects a file's magic bytes and extension to identify its format and category.
func Detect(filePath string) (*MediaInfo, error) {
	info := &MediaInfo{
		Path:           filePath,
		Format:         FormatUnknown,
		Category:       CategoryUnsupported,
		RequiresAction: false,
	}

	f, err := os.Open(filePath)
	if err != nil {
		return info, err
	}
	defer f.Close()

	// Read up to 512 bytes for signature detection
	header := make([]byte, 512)
	n, _ := io.ReadFull(f, header)
	header = header[:n]

	ext := strings.ToLower(filepath.Ext(filePath))

	// 1. Check Magic Bytes
	detectBySignature(header, ext, info)

	// 2. Fallback to extension if magic bytes were inconclusive or generic container
	if info.Format == FormatUnknown {
		detectByExtension(ext, info)
	}

	// 3. Mark actionable requirements
	switch info.Category {
	case CategoryVideo:
		info.RequiresAction = true
	case CategoryImage:
		// Standard JPEG is already compatible, other images need conversion to JPEG
		if info.Format == FormatJPEG {
			info.IsCompatible = true
			info.RequiresAction = false
			info.SkipReason = "File is already a JPEG image"
		} else {
			info.RequiresAction = true
		}
	case CategoryRAW, CategoryHEIC:
		info.RequiresAction = true
	}

	return info, nil
}

func detectBySignature(h []byte, ext string, info *MediaInfo) {
	if len(h) < 4 {
		return
	}

	// JPEG: FF D8 FF
	if len(h) >= 3 && h[0] == 0xFF && h[1] == 0xD8 && h[2] == 0xFF {
		info.Format = FormatJPEG
		info.Category = CategoryImage
		return
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if len(h) >= 8 && bytes.Equal(h[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		info.Format = FormatPNG
		info.Category = CategoryImage
		return
	}

	// GIF: GIF87a or GIF89a
	if len(h) >= 6 && (string(h[:6]) == "GIF87a" || string(h[:6]) == "GIF89a") {
		info.Format = FormatGIF
		info.Category = CategoryImage
		return
	}

	// BMP: 42 4D ("BM")
	if len(h) >= 2 && h[0] == 0x42 && h[1] == 0x4D {
		info.Format = FormatBMP
		info.Category = CategoryImage
		return
	}

	// Fujifilm RAF: "FUJIFILM"
	if len(h) >= 8 && string(h[:8]) == "FUJIFILM" {
		info.Format = FormatFujiRAF
		info.Category = CategoryRAW
		return
	}

	// Canon CR3: ISO BMFF with 'ftypcrx '
	if len(h) >= 12 && string(h[4:8]) == "ftyp" && strings.HasPrefix(string(h[8:12]), "crx") {
		info.Format = FormatCanonCR3
		info.Category = CategoryRAW
		return
	}

	// ISO BMFF container: MP4, MOV, HEIC
	if len(h) >= 12 && string(h[4:8]) == "ftyp" {
		brand := strings.ToLower(string(h[8:12]))
		switch {
		case strings.Contains(brand, "heic"), strings.Contains(brand, "heix"),
			strings.Contains(brand, "heim"), strings.Contains(brand, "heis"),
			strings.Contains(brand, "mif1"), strings.Contains(brand, "msf1"):
			info.Format = FormatHEIC
			info.Category = CategoryHEIC
			return
		case strings.Contains(brand, "qt"):
			info.Format = FormatMOV
			info.Category = CategoryVideo
			return
		default:
			info.Format = FormatMP4
			info.Category = CategoryVideo
			return
		}
	}

	// AVI: RIFF....AVI
	if len(h) >= 12 && string(h[:4]) == "RIFF" && string(h[8:12]) == "AVI " {
		info.Format = FormatAVI
		info.Category = CategoryVideo
		return
	}

	// Matroska / WebM: 1A 45 DF A3
	if len(h) >= 4 && bytes.Equal(h[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}) {
		if ext == ".webm" {
			info.Format = FormatWebM
		} else {
			info.Format = FormatMKV
		}
		info.Category = CategoryVideo
		return
	}

	// MPEG PS: 00 00 01 BA
	if len(h) >= 4 && bytes.Equal(h[:4], []byte{0x00, 0x00, 0x01, 0xBA}) {
		info.Format = FormatMPEG
		info.Category = CategoryVideo
		return
	}

	// AVCHD MTS / M2TS: MPEG-TS sync byte 0x47
	if (len(h) >= 188 && h[0] == 0x47 && h[188] == 0x47) ||
		(len(h) >= 192 && h[4] == 0x47 && h[196] == 0x47) ||
		ext == ".mts" || ext == ".m2ts" {
		info.Format = FormatMTS
		info.Category = CategoryVideo
		return
	}

	// TIFF and TIFF-based RAW (CR2, NEF, ARW, DNG)
	isLittleEndianTIFF := len(h) >= 4 && bytes.Equal(h[:4], []byte{0x49, 0x49, 0x2A, 0x00})
	isBigEndianTIFF := len(h) >= 4 && bytes.Equal(h[:4], []byte{0x4D, 0x4D, 0x00, 0x2A})

	if isLittleEndianTIFF || isBigEndianTIFF {
		// Canon CR2 has "CR" (0x43, 0x52) at offset 8, 9
		if len(h) >= 10 && h[8] == 0x43 && h[9] == 0x52 {
			info.Format = FormatCanonCR2
			info.Category = CategoryRAW
			return
		}

		// Differentiate other TIFF-based RAW by extension
		switch ext {
		case ".cr2":
			info.Format = FormatCanonCR2
			info.Category = CategoryRAW
		case ".nef":
			info.Format = FormatNikonNEF
			info.Category = CategoryRAW
		case ".arw":
			info.Format = FormatSonyARW
			info.Category = CategoryRAW
		case ".dng":
			info.Format = FormatAdobeDNG
			info.Category = CategoryRAW
		default:
			info.Format = FormatTIFF
			info.Category = CategoryImage
		}
		return
	}
}

func detectByExtension(ext string, info *MediaInfo) {
	switch ext {
	case ".avi":
		info.Format = FormatAVI
		info.Category = CategoryVideo
	case ".mov":
		info.Format = FormatMOV
		info.Category = CategoryVideo
	case ".mts", ".m2ts":
		info.Format = FormatMTS
		info.Category = CategoryVideo
	case ".mkv":
		info.Format = FormatMKV
		info.Category = CategoryVideo
	case ".webm":
		info.Format = FormatWebM
		info.Category = CategoryVideo
	case ".mpeg", ".mpg":
		info.Format = FormatMPEG
		info.Category = CategoryVideo
	case ".mp4":
		info.Format = FormatMP4
		info.Category = CategoryVideo
	case ".jpg", ".jpeg":
		info.Format = FormatJPEG
		info.Category = CategoryImage
	case ".png":
		info.Format = FormatPNG
		info.Category = CategoryImage
	case ".bmp":
		info.Format = FormatBMP
		info.Category = CategoryImage
	case ".tif", ".tiff":
		info.Format = FormatTIFF
		info.Category = CategoryImage
	case ".gif":
		info.Format = FormatGIF
		info.Category = CategoryImage
	case ".heic", ".heif", ".hif":
		info.Format = FormatHEIC
		info.Category = CategoryHEIC
	case ".cr2":
		info.Format = FormatCanonCR2
		info.Category = CategoryRAW
	case ".cr3":
		info.Format = FormatCanonCR3
		info.Category = CategoryRAW
	case ".nef":
		info.Format = FormatNikonNEF
		info.Category = CategoryRAW
	case ".arw":
		info.Format = FormatSonyARW
		info.Category = CategoryRAW
	case ".raf":
		info.Format = FormatFujiRAF
		info.Category = CategoryRAW
	case ".dng":
		info.Format = FormatAdobeDNG
		info.Category = CategoryRAW
	default:
		info.Format = FormatUnknown
		info.Category = CategoryUnsupported
	}
}
