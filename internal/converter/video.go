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

// VideoConverter handles video re-encoding using FFmpeg.
type VideoConverter struct{}

// NewVideoConverter creates a new video converter.
func NewVideoConverter() *VideoConverter {
	return &VideoConverter{}
}

func (v *VideoConverter) CanConvert(info *detector.MediaInfo) bool {
	return info.Category == detector.CategoryVideo
}

// FindFFmpegBinary locates ffmpeg in custom path, bundled app folder, or system PATH.
func FindFFmpegBinary(configuredPath string) string {
	if configuredPath != "" {
		if _, err := os.Stat(configuredPath); err == nil {
			return configuredPath
		}
	}

	if execPath, err := os.Executable(); err == nil {
		appDir := filepath.Dir(execPath)
		candidates := []string{
			filepath.Join(appDir, "ffmpeg", "ffmpeg.exe"),
			filepath.Join(appDir, "ffmpeg.exe"),
			filepath.Join(appDir, "ffmpeg", "ffmpeg"),
			filepath.Join(appDir, "ffmpeg"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path
	}

	return "ffmpeg"
}

// Convert executes FFmpeg to convert the input video to a phone-compatible MP4.
func (v *VideoConverter) Convert(ctx context.Context, sourcePath, destPath string, info *detector.MediaInfo, opts *Options) error {
	ffmpegBin := FindFFmpegBinary(opts.FFmpegPath)

	crf := opts.VideoCRF
	if crf <= 0 {
		crf = 20
	}

	preset := opts.VideoPreset
	if preset == "" {
		preset = "fast"
	}

	audioBitrate := opts.AudioBitrate
	if audioBitrate == "" {
		audioBitrate = "192k"
	}

	args := []string{
		"-y",                 // Overwrite destination if exists
		"-i", sourcePath,     // Input file
		"-c:v", "libx264",    // Standard H.264 codec
		"-crf", strconv.Itoa(crf),
		"-preset", preset,
		"-pix_fmt", "yuv420p", // Guaranteed phone/browser playback compatibility
		"-movflags", "+faststart",
	}

	// Audio handling: re-encode to AAC if audio stream is present
	if info.HasAudio {
		args = append(args, "-c:a", "aac", "-b:a", audioBitrate)
	} else {
		args = append(args, "-an") // No audio
	}

	args = append(args, "-f", "mp4", destPath)

	cmd := exec.CommandContext(ctx, ffmpegBin, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := stderr.String()
		if len(errStr) > 500 {
			errStr = errStr[len(errStr)-500:]
		}
		return fmt.Errorf("ffmpeg error converting %s: %w (details: %s)", filepath.Base(sourcePath), err, strings.TrimSpace(errStr))
	}

	return nil
}
