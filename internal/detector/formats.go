package detector

// Category represents the broad media class.
type Category string

const (
	CategoryVideo       Category = "video"
	CategoryImage       Category = "image"
	CategoryRAW         Category = "raw"
	CategoryHEIC        Category = "heic"
	CategoryUnsupported Category = "unsupported"
)

// Format represents the specific format identified.
type Format string

const (
	// Video Formats
	FormatAVI   Format = "AVI"
	FormatMOV   Format = "MOV"
	FormatMTS   Format = "MTS"
	FormatMKV   Format = "MKV"
	FormatWebM  Format = "WebM"
	FormatMPEG  Format = "MPEG"
	FormatMP4   Format = "MP4"

	// Standard Image Formats
	FormatJPEG Format = "JPEG"
	FormatPNG  Format = "PNG"
	FormatBMP  Format = "BMP"
	FormatTIFF Format = "TIFF"
	FormatGIF  Format = "GIF"

	// HEIC/HEIF
	FormatHEIC Format = "HEIC"

	// Camera RAW Formats
	FormatCanonCR2 Format = "Canon CR2"
	FormatCanonCR3 Format = "Canon CR3"
	FormatNikonNEF Format = "Nikon NEF"
	FormatSonyARW  Format = "Sony ARW"
	FormatFujiRAF  Format = "Fujifilm RAF"
	FormatAdobeDNG Format = "Adobe DNG"

	FormatUnknown Format = "UNKNOWN"
)

// MediaInfo contains detection metadata for a file.
type MediaInfo struct {
	Path           string
	Format         Format
	Category       Category
	IsCompatible   bool   // True if video/image is already phone-compatible and can skip re-encode
	RequiresAction bool   // True if file should be processed into Converted/
	SkipReason     string // Reason if skipped
	MIMEType       string
	Width          int
	Height         int
	VideoCodec     string
	AudioCodec     string
	HasAudio       bool
}
