//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/mukkss/MediaFlow/frontend"
	"github.com/mukkss/MediaFlow/internal/app"
	"github.com/mukkss/MediaFlow/internal/config"
	"github.com/mukkss/MediaFlow/internal/logger"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func main() {
	// 1. Load user configuration
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// 2. Initialize structured logger
	_, _ = logger.Init(cfg.LogLevel)
	logger.Get().Info("MediaFlow Desktop starting", "input", cfg.InputDir, "workers", cfg.WorkerCount)

	// 3. Initialize engine bridge service
	service := app.NewAppService(cfg)

	// 4. Initialize Wails Desktop Application
	wailsApp := application.New(application.Options{
		Name:        "MediaFlow",
		Description: "Automatic Camera Media Converter",
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontend.Assets),
		},
	})

	// Bridge internal service events to Wails frontend event emitter
	service.SetEventEmitter(func(event string, data any) {
		wailsApp.Event.Emit(event, data)
	})

	// 5. Create Desktop Window
	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "MediaFlow — Camera Media Converter",
		Width:     980,
		Height:    700,
		MinWidth:  800,
		MinHeight: 520,
	})

	// 6. Close-to-Tray: Hide window when user closes it instead of terminating
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		window.Hide()
	})

	// 7. System Tray (Notification Area)
	tray := wailsApp.SystemTray.New()
	tray.SetLabel("MediaFlow")

	trayMenu := wailsApp.NewMenu()
	trayMenu.Add("Show Dashboard").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Open Converted Folder").OnClick(func(ctx *application.Context) {
		_ = service.OpenFolder(cfg.EffectiveOutputDir())
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit MediaFlow").OnClick(func(ctx *application.Context) {
		_ = service.Stop()
		wailsApp.Quit()
	})
	tray.SetMenu(trayMenu)

	// Auto-start monitoring if input directory is configured
	if cfg.InputDir != "" {
		_ = service.Start()
	}

	// Run desktop application loop
	if err := wailsApp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal desktop error: %v\n", err)
	}
}
