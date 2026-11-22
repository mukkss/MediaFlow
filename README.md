# MediaFlow

A lightweight, reliable, and completely offline desktop utility designed to monitor user-configured camera import folders and automatically convert imported media into phone-compatible formats.

---

## Features

- **Automated Folder Monitoring**: Uses event-driven filesystem monitoring (`fsnotify`) with periodic reconciliation scans to catch new camera imports.
- **File Stability Detection**: Detects ongoing camera or SD card transfers over USB, holding processing until files are completely written.
- **Universal Video Transcoding**: Automatically converts camera videos (AVI, MOV, AVCHD MTS/M2TS, MKV, WebM, MPEG) into Faststart H.264 MP4 with AAC audio for smooth playback on any mobile device.
- **Smart MP4 Passthrough**: Inspects existing `.mp4` camera recordings via FFprobe; if already encoded in standard H.264 + AAC, skips re-encoding to save hours of processing time and preserve original quality.
- **Standard Image Conversion**: Converts PNG, BMP, TIFF, and GIF (first frame) into high-quality JPEG (default 92%).
- **Camera RAW Development**: Supports Canon (`.cr2`, `.cr3`), Nikon (`.nef`), Sony (`.arw`), Fujifilm (`.raf`), and Adobe (`.dng`) with camera preview extraction and EXIF orientation preservation.
- **Camera 10-bit HEIF/HIF Support**: Converts camera 10-bit HEIF files (Sony, Canon, Nikon) and iPhone HEIC into JPEG.
- **Stateless Filesystem Deduplication**: Automatically skips files that have already been converted in the `Converted/` directory, eliminating database corruption risks.
- **Collision Protection**: Automatically prevents overwriting by appending `_1`, `_2` suffixes if identical file names exist.
- **Safe Staging**: Writes output to temporary files and atomically renames them only after verification. Originals are **never** modified or deleted.
- **Background System Tray Operation (Windows)**: Closing the main window hides it to the Windows notification area (`window.Hide()`), allowing conversion to continue in the background.

---

## Supported Formats

| Category | Input Formats | Converted Output |
|---|---|---|
| **Videos** | AVI, QuickTime MOV, AVCHD (.mts, .m2ts), Matroska MKV, WebM, MPEG (.mpg, .mpeg), MP4 | **H.264 MP4** (`yuv420p`, AAC audio, Faststart) |
| **Standard Images** | JPEG (passthrough), PNG, BMP, TIFF (.tif, .tiff), GIF (first frame) | **JPEG** (92% quality, orientation preserved) |
| **Camera RAW** | Canon CR2, Canon CR3, Nikon NEF, Sony ARW, Fujifilm RAF, Adobe DNG | **JPEG** (camera preview extraction + LibRaw fallback) |
| **HEIC / HIF** | Apple HEIC, Sony 10-bit HIF, Canon 10-bit HIF, Nikon 10-bit HIF | **JPEG** (libheif + FFmpeg fallback) |

---

## How to Run on Linux (Right Now)

On your Linux system (Pop!_OS/Ubuntu), MediaFlow runs in **CLI Engine mode**.

### 1. Test the Pre-built CLI Binary

A compiled 64-bit Linux binary is already located in `bin/mediaflow`. Run:

```bash
# Display help and options
./bin/mediaflow -help
```

### 2. Start Monitoring a Camera Folder

```bash
# Monitor a folder and output converted files to <path>/Converted
./bin/mediaflow -input ~/Pictures/Camera_Import

# Specify custom output folder, worker concurrency, and quality
./bin/mediaflow -input ~/Pictures/Camera_Import -output ~/Pictures/Phone_Ready -workers 4 -crf 20 -quality 95
```

### 3. Build from Source

```bash
# Run all unit and integration tests
make test

# Recompile the Linux CLI binary
make build-cli
```

---

## How to Install and Run on Windows (10 / 11)

MediaFlow includes a desktop graphical application powered by **Wails v3** and **Svelte**, as well as an **Inno Setup** installer script.

### Option A: Run the Standalone Windows Executable Directly

A cross-compiled 64-bit Windows binary is already built at `bin/MediaFlow.exe` (17 MB, frontend Svelte assets embedded):

1. Copy `bin/MediaFlow.exe` to your Windows PC (e.g. via USB or download from your GitHub repository).
2. Ensure `ffmpeg.exe` and `ffprobe.exe` are placed either in the same folder as `MediaFlow.exe` or inside an `ffmpeg/` subfolder.
3. Double-click `MediaFlow.exe`. The desktop dashboard will launch immediately.

---

### Option B: Build the Windows Installer (`.exe`)

To build the one-click installer (`MediaFlow_Setup_1.0.0.exe`) on a Windows machine:

#### Prerequisites on Windows:
- **Go 1.26+**: [golang.org](https://go.dev/dl/)
- **Node.js 18+**: [nodejs.org](https://nodejs.org/)
- **Inno Setup 6+**: [jrsoftware.org/isdl.php](https://jrsoftware.org/isdl.php)
- **FFmpeg 7.1 LGPL Static Build**: Download `ffmpeg-n7.1-latest-win64-lgpl-7.1.zip` from [BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds/releases).

#### Step-by-Step Build on Windows:

1. **Clone the repository on Windows**:
   ```cmd
   git clone https://github.com/mukkss/MediaFlow.git
   cd MediaFlow
   ```

2. **Place FFmpeg in resources**:
   Extract `ffmpeg.exe` and `ffprobe.exe` into:
   ```text
   MediaFlow\resources\ffmpeg\ffmpeg.exe
   MediaFlow\resources\ffmpeg\ffprobe.exe
   ```

3. **Run the automated build script**:
   Open PowerShell and run:
   ```powershell
   .\build\build.ps1
   ```
   *This automatically installs frontend packages, compiles Svelte via Vite, compiles `MediaFlow.exe`, and invokes Inno Setup (`ISCC.exe`).*

4. **Install**:
   The installer package will be output to:
   ```text
   bin\MediaFlow_Setup_1.0.0.exe
   ```
   Run `MediaFlow_Setup_1.0.0.exe` to install MediaFlow with Start Menu integration, optional desktop icon, and optional startup registration.

---

## Configuration & Logs

User settings are saved in standard JSON format:
- **Windows**: `%APPDATA%\MediaFlow\config.json`
- **Linux**: `~/.config/MediaFlow/config.json`

Log files are automatically rotated (10 MB cap with 3 backups):
- **Windows**: `%APPDATA%\MediaFlow\logs\mediaflow.log`
- **Linux**: `~/.config/MediaFlow/logs/mediaflow.log`

---

## Third-Party Licensing

- MediaFlow is licensed under the MIT License.
- Video transcoding utilizes external `ffmpeg.exe` and `ffprobe.exe` binaries distributed under the **GNU Lesser General Public License (LGPL)**. MediaFlow communicates with FFmpeg strictly across standard operating system process boundaries. See [`resources/licenses/FFMPEG_LICENSE_COMPLIANCE.md`](resources/licenses/FFMPEG_LICENSE_COMPLIANCE.md) for full compliance details.
