package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Config holds user-configurable options for MediaFlow.
type Config struct {
	InputDir                string `json:"input_dir"`
	OutputDir               string `json:"output_dir"` // If empty, defaults to InputDir/Converted
	WorkerCount             int    `json:"worker_count"`
	VideoCRF                int    `json:"video_crf"`
	VideoPreset             string `json:"video_preset"`
	AudioBitrate            string `json:"audio_bitrate"`
	JPEGQuality             int    `json:"jpeg_quality"`
	StabilityIntervalMs     int    `json:"stability_interval_ms"`
	StabilityRequiredChecks int    `json:"stability_required_checks"`
	ScanIntervalSeconds     int    `json:"scan_interval_seconds"`
	LogLevel                string `json:"log_level"`
	FFmpegPath              string `json:"ffmpeg_path,omitempty"`
	FFprobePath             string `json:"ffprobe_path,omitempty"`
}

// DefaultConfig returns default production settings.
func DefaultConfig() *Config {
	return &Config{
		InputDir:                "",
		OutputDir:               "",
		WorkerCount:             2,
		VideoCRF:                20,
		VideoPreset:             "fast",
		AudioBitrate:            "192k",
		JPEGQuality:             92,
		StabilityIntervalMs:     2000,
		StabilityRequiredChecks: 3,
		ScanIntervalSeconds:     30,
		LogLevel:                "info",
	}
}

// GetAppDataDir returns the appropriate application data directory for the OS.
func GetAppDataDir() (string, error) {
	var baseDir string
	if runtime.GOOS == "windows" {
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			userProfile := os.Getenv("USERPROFILE")
			if userProfile != "" {
				baseDir = filepath.Join(userProfile, "AppData", "Roaming")
			}
		}
	}
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if runtime.GOOS == "windows" {
			baseDir = filepath.Join(home, "AppData", "Roaming")
		} else {
			baseDir = filepath.Join(home, ".config")
		}
	}

	appDir := filepath.Join(baseDir, "MediaFlow")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return appDir, nil
}

// GetConfigPath returns the canonical path to config.json.
func GetConfigPath() (string, error) {
	appDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "config.json"), nil
}

// Load loads the configuration from disk, creating default if not exists.
func Load() (*Config, error) {
	cfgPath, err := GetConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	cfg := DefaultConfig()
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = cfg.Save()
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	cfg.Validate()
	return cfg, nil
}

// Save writes the current configuration to disk.
func (c *Config) Save() error {
	cfgPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cfgPath, data, 0644)
}

// Validate ensures sane bounds on configuration values.
func (c *Config) Validate() {
	if c.WorkerCount < 1 {
		c.WorkerCount = 1
	} else if c.WorkerCount > 16 {
		c.WorkerCount = 16
	}

	if c.VideoCRF < 0 || c.VideoCRF > 51 {
		c.VideoCRF = 20
	}

	if c.VideoPreset == "" {
		c.VideoPreset = "fast"
	}

	if c.AudioBitrate == "" {
		c.AudioBitrate = "192k"
	}

	if c.JPEGQuality < 1 || c.JPEGQuality > 100 {
		c.JPEGQuality = 92
	}

	if c.StabilityIntervalMs < 500 {
		c.StabilityIntervalMs = 500
	}

	if c.StabilityRequiredChecks < 1 {
		c.StabilityRequiredChecks = 3
	}

	if c.ScanIntervalSeconds < 5 {
		c.ScanIntervalSeconds = 30
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

// EffectiveOutputDir returns the output directory, falling back to InputDir/Converted if empty.
func (c *Config) EffectiveOutputDir() string {
	if c.OutputDir != "" {
		return c.OutputDir
	}
	if c.InputDir != "" {
		return filepath.Join(c.InputDir, "Converted")
	}
	return ""
}
