package watcher

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mukkss/MediaFlow/internal/logger"
)

// Watcher monitors directories using fsnotify and invokes callbacks on stable media events.
type Watcher struct {
	inputDir   string
	outputDir  string
	watcher    *fsnotify.Watcher
	stability  *StabilityChecker
	onFile     func(path string)
	recentSeen map[string]time.Time
	mu         sync.Mutex
}

// NewWatcher initializes a filesystem watcher.
func NewWatcher(inputDir, outputDir string, intervalMs, requiredChecks int, onFile func(path string)) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		inputDir:   inputDir,
		outputDir:  outputDir,
		watcher:    fw,
		stability:  NewStabilityChecker(intervalMs, requiredChecks),
		onFile:     onFile,
		recentSeen: make(map[string]time.Time),
	}, nil
}

// Start begins watching the directory tree recursively until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	if err := w.watchRecursively(w.inputDir); err != nil {
		return err
	}

	go func() {
		defer w.watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				w.handleEvent(ctx, event)
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				logger.Get().Error("fsnotify error", "err", err)
			}
		}
	}()

	return nil
}

func (w *Watcher) handleEvent(ctx context.Context, event fsnotify.Event) {
	path := event.Name

	// Ignore output folder and hidden paths
	cleanPath, _ := filepath.Abs(path)
	cleanOutput, _ := filepath.Abs(w.outputDir)
	if cleanPath == cleanOutput || strings.HasPrefix(cleanPath, cleanOutput+string(filepath.Separator)) {
		return
	}

	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, ".tmp") || strings.EqualFold(base, "Converted") {
		return
	}

	// If a new directory was created, add it to the watcher
	if event.Has(fsnotify.Create) {
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			_ = w.watchRecursively(path)
			return
		}
	}

	// Trigger on file creation or write
	if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
		w.mu.Lock()
		last, seen := w.recentSeen[path]
		// Debounce duplicate events within 1 second
		if seen && time.Since(last) < 1*time.Second {
			w.mu.Unlock()
			return
		}
		w.recentSeen[path] = time.Now()
		w.mu.Unlock()

		// Launch stability check in background
		go func(filePath string) {
			if w.stability.WaitUntilStable(ctx, filePath) {
				if w.onFile != nil {
					w.onFile(filePath)
				}
			}
		}(path)
	}
}

func (w *Watcher) watchRecursively(root string) error {
	cleanOutput, _ := filepath.Abs(w.outputDir)

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			absPath, _ := filepath.Abs(path)
			if absPath == cleanOutput || strings.HasPrefix(absPath, cleanOutput+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			if strings.HasPrefix(d.Name(), ".") || strings.EqualFold(d.Name(), "Converted") {
				return filepath.SkipDir
			}
			if err := w.watcher.Add(path); err != nil {
				logger.Get().Warn("Failed to add watch on directory", "dir", path, "err", err)
			} else {
				logger.Get().Debug("Watching directory", "dir", path)
			}
		}
		return nil
	})
}
