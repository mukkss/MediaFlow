package detector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type probeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
}

// FindFFprobeBinary searches for ffprobe in bundled directory, config override, or system PATH.
func FindFFprobeBinary(configuredPath string) string {
	if configuredPath != "" {
		if _, err := os.Stat(configuredPath); err == nil {
			return configuredPath
		}
	}

	// Check relative to executable (bundled package location)
	if execPath, err := os.Executable(); err == nil {
		appDir := filepath.Dir(execPath)
		candidates := []string{
			filepath.Join(appDir, "ffmpeg", "ffprobe.exe"),
			filepath.Join(appDir, "ffprobe.exe"),
			filepath.Join(appDir, "ffmpeg", "ffprobe"),
			filepath.Join(appDir, "ffprobe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	// Fallback to PATH lookup
	if path, err := exec.LookPath("ffprobe"); err == nil {
		return path
	}

	return "ffprobe"
}

// InspectVideo uses ffprobe to inspect video codecs, streams, and compatibility.
func InspectVideo(ctx context.Context, ffprobeBin string, info *MediaInfo) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobeBin,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		info.Path,
	)

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ffprobe failed on %s: %w", info.Path, err)
	}

	var parsed probeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return fmt.Errorf("failed to parse ffprobe json: %w", err)
	}

	info.HasAudio = false
	info.VideoCodec = ""
	info.AudioCodec = ""

	for _, s := range parsed.Streams {
		if s.CodecType == "video" && info.VideoCodec == "" {
			info.VideoCodec = strings.ToLower(s.CodecName)
			info.Width = s.Width
			info.Height = s.Height
		} else if s.CodecType == "audio" && info.AudioCodec == "" {
			info.AudioCodec = strings.ToLower(s.CodecName)
			info.HasAudio = true
		}
	}

	// Compatibility evaluation for MP4 container files:
	// If MP4 container already has h264 video and aac (or no) audio, mark as compatible.
	if info.Format == FormatMP4 {
		isH264 := info.VideoCodec == "h264" || info.VideoCodec == "avc1"
		isAACorNone := !info.HasAudio || info.AudioCodec == "aac"

		if isH264 && isAACorNone {
			info.IsCompatible = true
			info.RequiresAction = false
			info.SkipReason = "MP4 is already phone-compatible (H.264/AAC)"
		} else {
			info.IsCompatible = false
			info.RequiresAction = true
		}
	}

	return nil
}
