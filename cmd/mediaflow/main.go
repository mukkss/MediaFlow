package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mukkss/MediaFlow/internal/config"
	"github.com/mukkss/MediaFlow/internal/converter"
	"github.com/mukkss/MediaFlow/internal/logger"
	"github.com/mukkss/MediaFlow/internal/output"
	"github.com/mukkss/MediaFlow/internal/queue"
	"github.com/mukkss/MediaFlow/internal/watcher"
)

func main() {
	inputFlag := flag.String("input", "", "Directory to monitor for camera media")
	outputFlag := flag.String("output", "", "Directory to save converted files (default: input/Converted)")
	workersFlag := flag.Int("workers", 0, "Number of concurrent conversion workers (default: 2)")
	crfFlag := flag.Int("crf", 0, "Video H.264 CRF quality factor (0-51, default: 20)")
	qualityFlag := flag.Int("quality", 0, "JPEG image quality (1-100, default: 92)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: failed to load config: %v (using defaults)\n", err)
		cfg = config.DefaultConfig()
	}

	// Apply CLI overrides if provided
	if *inputFlag != "" {
		cfg.InputDir = *inputFlag
	}
	if *outputFlag != "" {
		cfg.OutputDir = *outputFlag
	}
	if *workersFlag > 0 {
		cfg.WorkerCount = *workersFlag
	}
	if *crfFlag > 0 {
		cfg.VideoCRF = *crfFlag
	}
	if *qualityFlag > 0 {
		cfg.JPEGQuality = *qualityFlag
	}

	if cfg.InputDir == "" {
		fmt.Println("Error: Input directory is required. Specify via -input <path> or in config.json")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize Logger
	log, err := logger.Init(cfg.LogLevel)
	if err != nil {
		fmt.Printf("Warning initializing logger: %v\n", err)
	}

	log.Info("MediaFlow Engine starting",
		"inputDir", cfg.InputDir,
		"outputDir", cfg.EffectiveOutputDir(),
		"workers", cfg.WorkerCount,
		"videoCRF", cfg.VideoCRF,
		"jpegQuality", cfg.JPEGQuality,
	)

	// Verify input directory exists
	if _, err := os.Stat(cfg.InputDir); os.IsNotExist(err) {
		log.Error("Input directory does not exist", "path", cfg.InputDir)
		os.Exit(1)
	}

	// Initialize Output Manager
	outMgr := output.NewManager(cfg.InputDir, cfg.EffectiveOutputDir())
	outMgr.CleanStaleTemps()

	// Setup Options
	opts := &converter.Options{
		VideoCRF:     cfg.VideoCRF,
		VideoPreset:  cfg.VideoPreset,
		AudioBitrate: cfg.AudioBitrate,
		JPEGQuality:  cfg.JPEGQuality,
		FFmpegPath:   cfg.FFmpegPath,
		FFprobePath:  cfg.FFprobePath,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Queue
	q := queue.NewQueue(cfg.WorkerCount, outMgr, opts, func(job *queue.Job) {
		switch job.Status {
		case queue.StatusCompleted:
			log.Info("✅ Converted", "file", job.SourcePath, "dest", job.DestPath)
		case queue.StatusSkipped:
			log.Info("⏭️ Skipped", "file", job.SourcePath, "reason", job.Error)
		case queue.StatusFailed:
			log.Error("❌ Failed", "file", job.SourcePath, "err", job.Error)
		case queue.StatusUnsupported:
			log.Warn("⚠️ Unsupported", "file", job.SourcePath)
		}
	})
	q.Start(ctx)

	// Callback when a stable file is discovered
	onStableFile := func(filePath string) {
		q.Enqueue(ctx, filePath)
	}

	// Initialize Watcher
	fw, err := watcher.NewWatcher(cfg.InputDir, cfg.EffectiveOutputDir(), cfg.StabilityIntervalMs, cfg.StabilityRequiredChecks, onStableFile)
	if err != nil {
		log.Error("Failed to initialize folder watcher", "err", err)
		os.Exit(1)
	}

	if err := fw.Start(ctx); err != nil {
		log.Error("Failed to start folder watcher", "err", err)
		os.Exit(1)
	}

	// Initialize Periodic Scanner
	scanner := watcher.NewScanner(cfg.InputDir, cfg.EffectiveOutputDir(), cfg.ScanIntervalSeconds, onStableFile)
	go scanner.Start(ctx)

	log.Info("MediaFlow is monitoring for camera media. Press Ctrl+C to stop.")

	// Wait for OS shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Info("Shutting down MediaFlow gracefully...")
	cancel()
	q.Stop()

	stats := q.GetStats()
	log.Info("Final Session Stats",
		"detected", stats.TotalDetected,
		"converted", stats.TotalConverted,
		"skipped", stats.TotalSkipped,
		"failed", stats.TotalFailed,
		"unsupported", stats.TotalUnsupported,
	)
	log.Info("MediaFlow stopped.")
	time.Sleep(500 * time.Millisecond)
}
