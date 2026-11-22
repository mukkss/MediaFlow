package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInnoSetupScriptDirectives(t *testing.T) {
	issPath := filepath.Join("..", "installer", "setup.iss")
	content, err := os.ReadFile(issPath)
	if err != nil {
		t.Fatalf("failed to read setup.iss: %v", err)
	}

	str := string(content)

	// Requirement 5: Desktop shortcut optional (unchecked)
	if !strings.Contains(str, `Name: "desktopicon"`) || !strings.Contains(str, "Flags: unchecked") {
		t.Errorf("setup.iss missing unchecked desktopicon task")
	}

	// Requirement 6: Autostart optional and cleaned up on uninstallation
	if !strings.Contains(str, `Name: "autostart"`) {
		t.Errorf("setup.iss missing autostart task")
	}
	if !strings.Contains(str, "uninsdeletevalue") {
		t.Errorf("setup.iss missing uninsdeletevalue flag for autostart registry entry")
	}

	// Requirement 7: AppData preservation
	if !strings.Contains(str, "CurUninstallStepChanged") || !strings.Contains(str, "MediaFlow settings") {
		t.Errorf("setup.iss missing AppData confirmation dialog logic in uninstall step")
	}

	// Requirement 4: WebView2 check
	if !strings.Contains(str, "IsWebView2Installed") {
		t.Errorf("setup.iss missing WebView2 runtime check function")
	}
}

func TestPackagingArtifactsPresence(t *testing.T) {
	files := []string{
		filepath.Join("..", "installer", "setup.iss"),
		filepath.Join("..", "installer", "TEST_PROCEDURES.md"),
		filepath.Join("..", "build", "build.ps1"),
		filepath.Join("..", "Makefile"),
		filepath.Join("..", "resources", "licenses", "FFMPEG_LICENSE_COMPLIANCE.md"),
	}

	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("expected packaging artifact missing: %s", f)
		}
	}
}
