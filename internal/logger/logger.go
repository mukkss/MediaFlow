package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mukkss/MediaFlow/internal/config"
)

var (
	globalLogger *slog.Logger
	mu           sync.Mutex
)

// Init initializes the structured logger.
func Init(levelStr string) (*slog.Logger, error) {
	mu.Lock()
	defer mu.Unlock()

	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	appDir, err := config.GetAppDataDir()
	var logWriter io.Writer = os.Stdout

	if err == nil {
		logsDir := filepath.Join(appDir, "logs")
		_ = os.MkdirAll(logsDir, 0755)
		logFilePath := filepath.Join(logsDir, "mediaflow.log")

		// Rotate if file is larger than 10MB
		rotateLogIfNeeded(logFilePath, 10*1024*1024, 3)

		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			logWriter = io.MultiWriter(os.Stdout, file)
		}
	}

	handler := slog.NewTextHandler(logWriter, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339)),
				}
			}
			return a
		},
	})

	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)
	return globalLogger, nil
}

// Get returns the global logger or standard default if not initialized.
func Get() *slog.Logger {
	mu.Lock()
	defer mu.Unlock()
	if globalLogger == nil {
		globalLogger = slog.Default()
	}
	return globalLogger
}

func rotateLogIfNeeded(path string, maxSize int64, maxBackups int) {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < maxSize {
		return
	}

	// Rotate existing backups
	for i := maxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", path, i)
		newPath := fmt.Sprintf("%s.%d", path, i+1)
		_ = os.Rename(oldPath, newPath)
	}

	// Rename current log file to .1
	_ = os.Rename(path, fmt.Sprintf("%s.1", path))
}
