package watcher

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/mukkss/MediaFlow/internal/logger"
)

// Scanner performs periodic full-directory scans to reconcile files missed by fsnotify.
type Scanner struct {
	inputDir  string
	outputDir string
	interval  time.Duration
	onFile    func(path string)
}

// NewScanner creates a new reconciliation scanner.
func NewScanner(inputDir, outputDir string, intervalSeconds int, onFile func(path string)) *Scanner {
	if intervalSeconds < 5 {
		intervalSeconds = 30
	}
	return &Scanner{
		inputDir:  inputDir,
		outputDir: outputDir,
		interval:  time.Duration(intervalSeconds) * time.Second,
		onFile:    onFile,
	}
}

// Start runs the periodic scanner in the background until ctx is cancelled.
func (s *Scanner) Start(ctx context.Context) {
	// Run initial scan immediately
	s.ScanOnce()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.ScanOnce()
		}
	}
}

// ScanOnce performs a single pass over the input directory tree.
func (s *Scanner) ScanOnce() {
	if s.inputDir == "" {
		return
	}

	cleanInput, _ := filepath.Abs(s.inputDir)
	cleanOutput, _ := filepath.Abs(s.outputDir)

	_ = filepath.WalkDir(s.inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		absPath, _ := filepath.Abs(path)

		// Ignore output directory and subdirectories
		if absPath == cleanOutput || strings.HasPrefix(absPath, cleanOutput+string(filepath.Separator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Ignore hidden directories (.git, .tmp) or Converted directory
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || strings.EqualFold(name, "Converted") {
				return filepath.SkipDir
			}
			return nil
		}

		// Ignore temp or hidden files
		name := d.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".tmp") {
			return nil
		}

		// Don't scan the root folder as a file
		if absPath == cleanInput {
			return nil
		}

		if s.onFile != nil {
			s.onFile(path)
		}

		return nil
	})

	logger.Get().Debug("Reconciliation scan completed", "inputDir", s.inputDir)
}
