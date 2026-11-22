# MediaFlow — FFmpeg & Third-Party Licensing Compliance Specification

**Document Version:** 1.0.0  
**Target Platform:** Windows 10/11 (64-bit)  
**Distribution Model:** Bundled Offline Desktop Application

---

## 1. FFmpeg Licensing Analysis: LGPL vs. GPL

MediaFlow redistributes `ffmpeg.exe` and `ffprobe.exe` alongside the application. To ensure full legal compliance, the distribution boundaries and licensing requirements must be documented and maintained.

### 1.1 The Crucial Distinction

FFmpeg is natively licensed under the **GNU Lesser General Public License (LGPL) version 2.1 or later**. However, certain external optional codecs and libraries trigger the **GNU General Public License (GPL) version 2 or 3**:

| Component | Default License | Impact on Compiled Binary |
|---|---|---|
| FFmpeg Core (`libavcodec`, `libavformat`, `libavutil`, `libswscale`) | **LGPL-2.1+** | Permits distribution with proprietary/permissive code if dynamically linked or invoked across a CLI process boundary. |
| `libx264` (H.264 software encoder) | **GPL-2.0+** | **Forces the entire compiled binary under GPL.** |
| `libx265` (HEVC software encoder) | **GPL-2.0+** | **Forces the entire compiled binary under GPL.** |
| `libxvid` (MPEG-4 Part 2 encoder) | **GPL-2.0+** | **Forces the entire compiled binary under GPL.** |
| `libopenh264` (Cisco H.264 encoder/decoder) | **BSD 2-Clause** | **LGPL-compatible.** |
| Native Windows Media Foundation (`h264_mf`) | **System OS API** | **LGPL-compatible** (no GPL contamination). |
| Native FFmpeg AAC encoder (`aac`) | **LGPL-2.1+** | **LGPL-compatible.** |

### 1.2 Selected Binary Variant: BtbN LGPL Static Build

To strictly satisfy the requirement of an **LGPL distribution**, MediaFlow requires the **LGPL static release build**:
- **Source Repository:** [BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds/releases)
- **Target Archive:** `ffmpeg-n7.1-latest-win64-lgpl-7.1.zip` (or `ffmpeg-master-latest-win64-lgpl.zip`)
- **Build Configuration Flags:**
  ```text
  --disable-gpl --disable-nonfree --enable-version3 --enable-libopenh264 --enable-libopus --enable-libvpx --enable-libvorbis
  ```
- **Codec Verification:**
  - Video Encoders: `libopenh264`, `h264_mf` (Windows Media Foundation hardware/software), `mpeg4`, `mjpeg`
  - Audio Encoders: `aac` (FFmpeg native), `opus`, `pcm_s16le`
  - Video Decoders: All standard formats (H.264, HEVC, MPEG-2, MPEG-4, Motion JPEG, ProRes, VP8, VP9, AV1)

### 1.3 Subprocess Isolation & License Boundary

MediaFlow does **not** link against FFmpeg C libraries via CGO (`libavcodec-dev`).  
Instead, MediaFlow executes `ffmpeg.exe` and `ffprobe.exe` as **independent external child processes** via Go's `os/exec` command pipeline.

Under Free Software Foundation (FSF) and legal guidelines:
1. Invoking an external binary via command-line arguments and standard I/O pipes establishes an **arms-length process boundary**.
2. MediaFlow's proprietary and core Go codebase is **not a derivative work** of FFmpeg.
3. MediaFlow's installer aggregates and distributes FFmpeg as an independent utility package.

### 1.4 Redistribution Obligations Under LGPL-2.1 / LGPL-3.0

When distributing the precompiled FFmpeg binaries with MediaFlow, the installer and documentation must satisfy:
1. **Notice Provision:** Explicitly notify the user that FFmpeg is used and that it is licensed under the LGPL (included in the installer, `About` dialog, and `licenses/` directory).
2. **License Text:** Include the full text of `LGPL-2.1.txt` and `LGPL-3.0.txt` in `{app}\licenses\`.
3. **Source Code Offer:** Provide written instructions or a download link to obtain the exact matching source code and build scripts of the distributed FFmpeg release.

---

## 2. LibRaw & libheif Licensing & Runtime Dependencies

### 2.1 LibRaw
- **License:** Dual-licensed under **CDDL 1.0** and **LGPL 2.1**.
- **Packaging:** Distributed in `{app}\codecs\` as a standalone utility (`dcraw_emu.exe`) or dynamic library (`libraw.dll`).
- **Dependencies:** MinGW x64 runtime DLLs (`libwinpthread-1.dll`, `libgcc_s_seh-1.dll`, `libstdc++-6.dll`) or MSVC runtime (`vcruntime140.dll`).

### 2.2 libheif
- **License:** **LGPL-3.0+**.
- **Dependencies:** `libde265.dll` (LGPL).
- **Packaging:** Distributed in `{app}\codecs\` alongside license text `libheif-LGPL.txt`.

---

## 3. Microsoft Edge WebView2 Runtime Prerequisite

### 3.1 Availability Matrix
- **Windows 11:** Pre-installed on all editions.
- **Windows 10 (20H2 and newer):** Pre-installed via Microsoft cumulative monthly updates.
- **Offline / Windows 10 Enterprise LTSC:** May not have WebView2 installed.

### 3.2 Offline Installation Strategy
Inno Setup checks registry key:
`HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
1. If found, proceed without interruption.
2. If absent and `redist/MicrosoftEdgeWebView2RuntimeInstallerX64.exe` is present in the installer directory, launch silent local installation.
3. If absent and no local installer is bundled, alert user with manual download instructions rather than hanging.
