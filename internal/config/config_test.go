package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.WorkerCount != 2 {
		t.Errorf("expected WorkerCount 2, got %d", cfg.WorkerCount)
	}
	if cfg.VideoCRF != 20 {
		t.Errorf("expected VideoCRF 20, got %d", cfg.VideoCRF)
	}
	if cfg.JPEGQuality != 92 {
		t.Errorf("expected JPEGQuality 92, got %d", cfg.JPEGQuality)
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := &Config{
		WorkerCount: 99,
		VideoCRF:    99,
		JPEGQuality: 150,
	}
	cfg.Validate()

	if cfg.WorkerCount != 16 {
		t.Errorf("expected bounded WorkerCount 16, got %d", cfg.WorkerCount)
	}
	if cfg.VideoCRF != 20 {
		t.Errorf("expected default VideoCRF 20, got %d", cfg.VideoCRF)
	}
	if cfg.JPEGQuality != 92 {
		t.Errorf("expected bounded JPEGQuality 92, got %d", cfg.JPEGQuality)
	}
}

func TestEffectiveOutputDir(t *testing.T) {
	cfg := &Config{
		InputDir: "/path/to/Camera_Import",
	}
	expected := filepath.Join("/path/to/Camera_Import", "Converted")
	if cfg.EffectiveOutputDir() != expected {
		t.Errorf("expected %s, got %s", expected, cfg.EffectiveOutputDir())
	}

	cfg.OutputDir = "/custom/Converted"
	if cfg.EffectiveOutputDir() != "/custom/Converted" {
		t.Errorf("expected /custom/Converted, got %s", cfg.EffectiveOutputDir())
	}
}
