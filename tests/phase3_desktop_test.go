package tests

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mukkss/MediaFlow/frontend"
	"github.com/mukkss/MediaFlow/internal/app"
	"github.com/mukkss/MediaFlow/internal/config"
)

func TestEmbeddedFrontendAssets(t *testing.T) {
	// 1. Verify index.html exists in embedded assets
	f, err := frontend.Assets.Open("dist/index.html")
	if err != nil {
		t.Fatalf("failed to open embedded dist/index.html: %v", err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read embedded index.html: %v", err)
	}

	htmlStr := string(content)
	if !strings.Contains(htmlStr, "<div id=\"app\">") {
		t.Errorf("embedded index.html does not contain app mount container: %s", htmlStr)
	}
}

func TestAppServiceFrontendBindings(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "Import")
	outputDir := filepath.Join(tmpDir, "Export")
	_ = os.MkdirAll(inputDir, 0755)

	cfg := &config.Config{
		InputDir:    inputDir,
		OutputDir:   outputDir,
		WorkerCount: 3,
		VideoCRF:    20,
		JPEGQuality: 92,
	}

	service := app.NewAppService(cfg)

	// 1. Check GetConfig
	c := service.GetConfig()
	if c.WorkerCount != 3 || c.InputDir != inputDir {
		t.Errorf("unexpected config returned: %+v", c)
	}

	// 2. Check Event Emitter
	var receivedEvent string
	service.SetEventEmitter(func(event string, data any) {
		receivedEvent = event
	})

	if err := service.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer service.Stop()

	if receivedEvent != "monitoring_state" {
		t.Errorf("expected monitoring_state event, got: %s", receivedEvent)
	}
}
