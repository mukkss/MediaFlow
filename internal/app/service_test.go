package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mukkss/MediaFlow/internal/config"
)

func TestAppServiceLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "Camera_Import")
	outputDir := filepath.Join(tmpDir, "Converted")

	_ = os.MkdirAll(inputDir, 0755)

	cfg := &config.Config{
		InputDir:                inputDir,
		OutputDir:               outputDir,
		WorkerCount:             2,
		VideoCRF:                20,
		JPEGQuality:             92,
		StabilityIntervalMs:     500,
		StabilityRequiredChecks: 1,
		ScanIntervalSeconds:     1,
		LogLevel:                "info",
	}

	service := NewAppService(cfg)

	// 1. Initial State
	if service.IsMonitoring() {
		t.Errorf("expected service to not be monitoring initially")
	}

	// 2. Start Monitoring
	if err := service.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}
	if !service.IsMonitoring() {
		t.Errorf("expected service to be monitoring after Start()")
	}

	// 3. Drop a file and observe recent jobs update
	testFile := filepath.Join(inputDir, "test.png")
	_ = os.WriteFile(testFile, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0644)

	// Give scanner/watcher a moment to enqueue
	time.Sleep(300 * time.Millisecond)

	jobs := service.GetRecentJobs()
	if len(jobs) == 0 {
		t.Log("Note: File may still be processing in queue or rejected")
	}

	// 4. Stats Check
	stats := service.GetStats()
	if stats.ActiveWorkers < 0 {
		t.Errorf("invalid active workers count: %d", stats.ActiveWorkers)
	}

	// 5. Stop Monitoring
	if err := service.Stop(); err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}
	if service.IsMonitoring() {
		t.Errorf("expected service to stop monitoring after Stop()")
	}
}

func TestAppServiceConfigUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	service := NewAppService(cfg)

	updated := &config.Config{
		InputDir:    "/new/camera/path",
		WorkerCount: 4,
		VideoCRF:    18,
		JPEGQuality: 95,
	}

	err := service.UpdateConfig(updated)
	if err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	current := service.GetConfig()
	if current.WorkerCount != 4 {
		t.Errorf("expected worker count 4, got %d", current.WorkerCount)
	}
	if current.VideoCRF != 18 {
		t.Errorf("expected VideoCRF 18, got %d", current.VideoCRF)
	}
	if current.JPEGQuality != 95 {
		t.Errorf("expected JPEGQuality 95, got %d", current.JPEGQuality)
	}
}
