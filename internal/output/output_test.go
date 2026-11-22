package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutputManager(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "Camera_Import")
	outputDir := filepath.Join(tmpDir, "Converted")

	_ = os.MkdirAll(inputDir, 0755)

	m := NewManager(inputDir, outputDir)

	// Test 1: Simple file path resolution
	src := filepath.Join(inputDir, "DSC_001.MOV")
	target := m.ResolveTargetPath(src, false, false)
	expected := filepath.Join(outputDir, "DSC_001.mp4")
	if target != expected {
		t.Errorf("expected %s, got %s", expected, target)
	}

	// Test 2: Nested folder structure preservation
	nestedSrc := filepath.Join(inputDir, "2026_Vacation", "Day1", "IMG_002.CR3")
	nestedTarget := m.ResolveTargetPath(nestedSrc, true, false)
	expectedNested := filepath.Join(outputDir, "2026_Vacation", "Day1", "IMG_002.jpg")
	if nestedTarget != expectedNested {
		t.Errorf("expected %s, got %s", expectedNested, nestedTarget)
	}

	// Test 3: Collision avoidance
	_ = os.MkdirAll(filepath.Dir(expected), 0755)
	_ = os.WriteFile(expected, []byte("existing"), 0644)

	collisionTarget := m.ResolveTargetPath(src, false, true)
	expectedCollision := filepath.Join(outputDir, "DSC_001_1.mp4")
	if collisionTarget != expectedCollision {
		t.Errorf("expected collision path %s, got %s", expectedCollision, collisionTarget)
	}

	// Test 4: Atomic finalization
	tempPath := m.TempPathFor(expectedCollision)
	if err := os.WriteFile(tempPath, []byte("converted data"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	if err := m.Finalize(tempPath, expectedCollision); err != nil {
		t.Fatalf("failed to finalize temp file: %v", err)
	}

	if _, err := os.Stat(expectedCollision); os.IsNotExist(err) {
		t.Errorf("finalized file does not exist")
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("temp file was not removed after finalization")
	}
}
