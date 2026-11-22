# MediaFlow — Linux Development & Windows Cross-Compilation Audit

**Audit Date:** 2026-11-22  
**Host Environment:** Pop!_OS 24.04 LTS (Linux x86_64, Kernel 6.9+)  
**Toolchain Versions:** Go 1.26.2, Node.js v24.15.0, npm v10.8.2  
**Target Architecture:** Windows 10/11 64-bit (`x86_64-pc-windows-msvc` / `windows/amd64`)

---

## 1. Executive Summary

This audit establishes the exact cross-compilation capabilities and boundaries of the MediaFlow repository when developed on Linux and packaged for Windows.

| Target Component | Source Entry Point | Host Compilation (Linux) | Target Compilation (Windows) | Verification Status |
|---|---|---|---|---|
| **Core Conversion Engine (CLI)** | `cmd/mediaflow/main.go` | `bin/mediaflow` (Linux ELF 64) | `bin/mediaflow.exe` (PE32+ x64) | **PASS** |
| **Desktop Application (GUI)** | `cmd/desktop/main.go` | Stubbed via `main_other.go` | `bin/MediaFlow.exe` (PE32+ x64 GUI) | **PASS** |
| **Svelte Frontend Assets** | `frontend/src/` | `frontend/dist/` (Static HTML/JS/CSS) | Embedded in `.exe` via `embed.FS` | **PASS** |
| **Inno Setup Installer** | `installer/setup.iss` | N/A (Windows `ISCC.exe` required) | `bin/MediaFlow_Setup_1.0.0.exe` | **Syntactically Validated** (Requires Windows execution) |

---

## 2. Cross-Compilation Breakdown

### 2.1 Core CLI Engine (`cmd/mediaflow`)
- **Compilation Command:**
  ```bash
  # Linux binary
  go build -o bin/mediaflow ./cmd/mediaflow
  # Windows binary
  GOOS=windows GOARCH=amd64 go build -o bin/mediaflow.exe ./cmd/mediaflow
  ```
- **Findings:**
  - The core engine has zero CGO dependencies for standard formats.
  - Generates a standalone native PE32+ executable for Windows with no external DLLs needed.

### 2.2 Desktop UI (`cmd/desktop`)
- **Compilation Command:**
  ```bash
  GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H windowsgui" -o bin/MediaFlow.exe ./cmd/desktop
  ```
- **Findings:**
  - Wails v3 relies on pure-Go Win32 syscalls and Windows WebView2 APIs when `GOOS=windows`.
  - It compiles cleanly from Linux to Windows without requiring a cross-C compiler (`x86_64-w64-mingw32-gcc`).
  - To prevent Linux test suite failures where Linux GTK4 headers (`gtk4`, `webkitgtk-6.0`) are not installed on Pop!_OS, `cmd/desktop/main.go` is guarded with `//go:build windows` and `cmd/desktop/main_other.go` provides the non-Windows stub.
  - Result: All Go unit and integration tests (`go test -v ./...`) run and pass on Linux without installing GTK4 development libraries.

### 2.3 Svelte Frontend (`frontend/`)
- **Build Command:**
  ```bash
  cd frontend && npm install && npm run build
  ```
- **Findings:**
  - Generates production-ready, minified bundle in `frontend/dist/` (HTML: 0.39 KB, JS: 20.37 KB, CSS: 5.11 KB).
  - Go's `//go:embed all:dist` embeds the files directly inside `MediaFlow.exe`.
  - Zero web server runtime, Node.js runtime, or browser process required by the end user on Windows.

---

## 3. Windows Native Packaging & Testing Boundary

### 3.1 What Has Been Executed & Verified on Linux:
1. Full test suite execution across all format converters (`tests/phase5_matrix_test.go`):
   - PNG, BMP, TIFF, GIF first frame to JPEG.
   - Canon CR2, Nikon NEF, Sony ARW (with embedded preview extraction) to JPEG.
   - AVI, MOV, MKV to H.264 MP4 via FFmpeg.
   - Faststart H.264+AAC MP4 detection and passthrough.
   - File stability detection on 0-byte and growing files.
   - 50-file concurrent burst import queueing.
   - Atomic temporary file finalization (`.tmp.mp4` / `.tmp.jpg`) and stale temporary file cleanup.
2. Inno Setup script validation (`tests/phase4_installer_test.go`):
   - Confirms optional desktop shortcut (`Flags: unchecked`).
   - Confirms optional Windows startup registration with `uninsdeletevalue` cleanup.
   - Confirms `%APPDATA%\MediaFlow\` preservation logic on uninstall.
   - Confirms WebView2 detection function.

### 3.2 What Requires a Windows Machine (Per `installer/TEST_PROCEDURES.md`):
1. **Compiling the Final Installer (`ISCC.exe`)**:
   Inno Setup compiler (`ISCC.exe`) is a Windows binary. Running `installer/setup.iss` or `build/build.ps1` must be performed on Windows (or via Wine with Inno Setup installed).
2. **Installation QA Testing**:
   Actual execution of `MediaFlow_Setup_1.0.0.exe` (fresh install, upgrade, uninstall, reinstallation, and system tray close-to-tray lifecycle) must be conducted on a real Windows 10/11 PC or VM as documented in `installer/TEST_PROCEDURES.md`.
