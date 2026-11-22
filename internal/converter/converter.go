package converter

import (
	"context"

	"github.com/mukkss/MediaFlow/internal/detector"
)

// Options holds runtime conversion parameters.
type Options struct {
	VideoCRF     int
	VideoPreset  string
	AudioBitrate string
	JPEGQuality  int
	FFmpegPath   string
	FFprobePath  string
}

// MediaConverter defines the interface implemented by category converters.
type MediaConverter interface {
	CanConvert(info *detector.MediaInfo) bool
	Convert(ctx context.Context, sourcePath, destPath string, info *detector.MediaInfo, opts *Options) error
}
