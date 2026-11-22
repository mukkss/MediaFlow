# MediaFlow — Windows Installer Quality Assurance & Validation Guide

This document defines the formal validation procedures to test `MediaFlow_Setup_1.0.0.exe` on clean Windows 10 and Windows 11 target environments.

---

## 1. Test Environment Matrix

| Test Node | Operating System | Build / Architecture | Dependencies Pre-installed |
|---|---|---|---|
| **VM 1** | Windows 11 Pro | 23H2 (x64) | None (Clean snapshot, no dev tools) |
| **VM 2** | Windows 10 Home/Pro | 22H2 (x64) | None (Standard OEM image with WebView2) |
| **VM 3** | Windows 10 Enterprise LTSC | 21H2 (x64) | Offline (Network adapter disabled, no WebView2) |

---

## 2. Test Cases and Procedures

### TC-01: Fresh Installation
**Objective:** Verify that MediaFlow installs without developer runtimes or environment configuration.
1. Copy `MediaFlow_Setup_1.0.0.exe` to clean VM 1.
2. Launch installer without administrative elevation (per-user lowest privilege mode).
3. **Verify:**
   - Default install path is `{autopf}\MediaFlow` (`C:\Users\<User>\AppData\Local\Programs\MediaFlow`).
   - Desktop icon checkbox is **unchecked** by default.
   - Autostart on boot checkbox is **unchecked** by default.
4. Complete installation.
5. Launch `MediaFlow` from the Windows Start Menu.
6. **Pass Criteria:** MediaFlow dashboard opens smoothly. Status shows "Monitoring Paused" or "Idle".

---

### TC-02: Optional Desktop Shortcut
1. Run installer. Check the "Create a desktop shortcut" option.
2. Complete installation.
3. **Pass Criteria:** A desktop icon appears pointing to `{app}\MediaFlow.exe`. Double-clicking launches the application.

---

### TC-03: Optional Windows Startup Registration & Cleanup
1. Run installer with "Automatically launch MediaFlow on Windows startup" enabled.
2. Open Windows Registry Editor (`regedit`).
3. Check `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`.
4. **Pass Criteria:** Value named `MediaFlow` exists pointing to `"{app}\MediaFlow.exe"`.
5. Run the Uninstaller.
6. Re-check the registry key.
7. **Pass Criteria:** The `MediaFlow` startup entry is automatically deleted upon uninstall.

---

### TC-04: User Data Preservation on Uninstall
1. Configure an input folder and convert sample media files.
2. Confirm configuration exists at `%APPDATA%\MediaFlow\config.json` and logs at `%APPDATA%\MediaFlow\logs\`.
3. Open Windows Settings → Apps → Installed Apps → Uninstall MediaFlow.
4. During uninstallation, when prompted: *"Do you want to delete your MediaFlow settings, conversion logs, and history from AppData?"*, click **No**.
5. **Pass Criteria:**
   - `{app}` installation directory is completely removed.
   - `%APPDATA%\MediaFlow\` remains intact.
6. Re-run installer (Reinstallation test) and launch MediaFlow.
7. **Pass Criteria:** Previous folder settings and log history are immediately restored.

---

### TC-05: Upgrade Installation
1. Install MediaFlow v1.0.0.
2. Build an updated setup package (e.g., v1.0.1).
3. Run the v1.0.1 installer without manually uninstalling v1.0.0.
4. **Pass Criteria:**
   - Overwrite occurs without file lock conflicts.
   - Previous configuration is preserved.
   - Version number in About tab updates to v1.0.1.

---

### TC-06: Offline WebView2 Detection
1. On VM 3 (offline without WebView2), run `MediaFlow_Setup_1.0.0.exe`.
2. **Pass Criteria:** Installer detects absence of WebView2 via registry check and displays a clear advisory dialog explaining the prerequisite.
