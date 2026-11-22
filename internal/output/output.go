package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager handles file naming, nested paths, atomic moves, and collisions.
type Manager struct {
	inputDir  string
	outputDir string
	mu        sync.Mutex
}

// NewManager creates an output Manager.
func NewManager(inputDir, outputDir string) *Manager {
	if outputDir == "" {
		outputDir = filepath.Join(inputDir, "Converted")
	}
	return &Manager{
		inputDir:  inputDir,
		outputDir: outputDir,
	}
}

// TargetExtension returns the target extension for a converted media category.
func TargetExtension(isImage bool) string {
	if isImage {
		return ".jpg"
	}
	return ".mp4"
}

// DestinationExists checks if the converted destination file already exists and is valid.
func (m *Manager) DestinationExists(sourcePath string, isImage bool) bool {
	destPath := m.ResolveTargetPath(sourcePath, isImage, false)
	fi, err := os.Stat(destPath)
	if err != nil {
		return false
	}
	// If file exists and size is greater than 0, consider already converted
	return fi.Size() > 0
}

// ResolveTargetPath computes the final destination path, preserving relative subdirectories.
func (m *Manager) ResolveTargetPath(sourcePath string, isImage bool, avoidCollision bool) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	relDir := ""
	if rel, err := filepath.Rel(m.inputDir, filepath.Dir(sourcePath)); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		relDir = rel
	}

	targetDir := filepath.Join(m.outputDir, relDir)
	_ = os.MkdirAll(targetDir, 0755)

	baseName := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	targetExt := TargetExtension(isImage)
	targetFileName := baseName + targetExt
	finalPath := filepath.Join(targetDir, targetFileName)

	if !avoidCollision {
		return finalPath
	}

	// Collision avoidance: DSC_001.mp4 -> DSC_001_1.mp4 -> DSC_001_2.mp4
	counter := 1
	for {
		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			break
		}
		targetFileName = fmt.Sprintf("%s_%d%s", baseName, counter, targetExt)
		finalPath = filepath.Join(targetDir, targetFileName)
		counter++
	}

	return finalPath
}

// TempPathFor returns a temporary staging file path for atomic writing.
func (m *Manager) TempPathFor(finalPath string) string {
	dir := filepath.Dir(finalPath)
	base := filepath.Base(finalPath)
	tempName := fmt.Sprintf(".%s.%d_%d.tmp", base, os.Getpid(), time.Now().UnixNano())
	return filepath.Join(dir, tempName)
}

// Finalize moves the temp file to finalPath atomically after verification.
func (m *Manager) Finalize(tempPath, finalPath string) error {
	fi, err := os.Stat(tempPath)
	if err != nil {
		return fmt.Errorf("temporary file missing before finalization: %w", err)
	}

	if fi.Size() == 0 {
		_ = os.Remove(tempPath)
		return fmt.Errorf("conversion generated an empty (0 byte) file")
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	// Atomic rename
	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to atomically finalize %s to %s: %w", tempPath, finalPath, err)
	}

	return nil
}

// CleanStaleTemps removes leftover temporary files from crashed or aborted sessions.
func (m *Manager) CleanStaleTemps() {
	_ = filepath.Walk(m.outputDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".tmp") {
			_ = os.Remove(path)
		}
		return nil
	})
}
