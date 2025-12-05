# WOL Manager

[中文](README.md) | [English](README_EN.md)

A simple yet powerful Wake-on-LAN (WOL) management tool.

### Features

*   **Modern Web Interface**: A responsive UI based on Bootstrap 5 with a dark theme (Atom One Dark style), featuring card-based device display.
*   **Smart Broadcast**:
    *   Supports specifying a broadcast IP.
    *   **Auto-Discovery**: If the broadcast IP is left empty, the system automatically iterates through all IPv4 network interfaces to send Magic Packets, ensuring successful wake-up even in multi-NIC environments.
*   **Log System**:
    *   Detailed records of wake-up operations, device management, and system errors.
    *   Real-time log viewing filtered by device.
    *   Automatic log rotation and cleanup (default retention: 3 days).
*   **Device Management**: Easily add, edit, and delete devices. Supports grouping multiple devices under one card.
*   **One-Click Wake**: Send Magic Packets with a single click (defaults to sending 5 times consecutively to ensure reliability).
*   **Status Monitoring**: Automatically detects device online status via ICMP Ping (🟢 Online / 🔴 Offline).
*   **Configuration Persistence**: All settings (port, device list, log settings) are stored in `wol.json` for easy migration and backup.
*   **Cross-Platform Support**:
    *   **Windows**: Minimizes to the system tray with a context menu (Open Web Page, Run at Startup, Exit).
    *   **Linux**:
        *   Supports `-d` flag to run as a daemon.
        *   Supports `-k` flag to stop the running daemon.
*   **CI/CD**: Integrated GitHub Actions workflow for automatic building on push.

### Download & Installation

#### 1. Get Source Code

```bash
git clone https://github.com/MajoSissi/wol.git
cd wol
```

#### 2. Install Dependencies

This project requires a Go environment.

*   **Go**: Ensure [Go 1.23+](https://go.dev/dl/) is installed.
*   **Linux Extra Dependencies**: GTK development libraries are required for the system tray.
    *   Ubuntu/Debian: `sudo apt-get install libgtk-3-dev libappindicator3-dev`
    *   CentOS/RHEL: `sudo yum install gtk3-devel libappindicator-gtk3-devel`

#### 3. Build

Run the following commands in the project root. Artifacts will be generated in the `build/` directory.

**Windows (PowerShell)**
```powershell
mkdir build
$env:GOOS='windows'; $env:GOARCH='amd64'; go build -ldflags "-H=windowsgui" -o build/wol-windows.exe
```

**Linux (Bash)**
```bash
mkdir -p build
GOOS=linux GOARCH=amd64 go build -o build/wol-linux
```

### Usage

#### Windows
1.  Run `build/wol-windows.exe`.
2.  The program will minimize to the system tray.
3.  **Right-click Tray Icon**:
    *   **Open Web Page**: Opens the management interface (default: http://localhost:8888).
    *   **Run at Startup**: Toggle to enable/disable auto-start.
    *   **Exit**: Quit the application.

#### Linux
*   **Foreground**: `./wol`
*   **Daemon**: `./wol -d`
*   **Stop Daemon**: `./wol -k`

### Configuration (`wol.json`)

Generated automatically on first run.

```json
{
  "port": 8888,
  "log_dir": "./logs",
  "log_retention_days": 3,
  "devices": [
    {
      "name": "Home Server",
      "sub_devices": [
        {
          "mac": "D4:5D:64:A1:B2:C3",
          "ip": "192.168.50.10",
          "port": 9,
          "broadcast_ip": "192.168.50.255",
          "remark": "Main Interface"
        }
      ]
    },
    {
      "name": "Office PCs",
      "sub_devices": [
        {
          "mac": "00:E0:4C:68:00:01",
          "ip": "10.0.1.101",
          "port": 9,
          "broadcast_ip": "",
          "remark": "Workstation 01"
        },
        {
          "mac": "00:E0:4C:68:00:02",
          "ip": "10.0.1.102",
          "port": 9,
          "broadcast_ip": "",
          "remark": "Workstation 02"
        }
      ]
    }
  ]
}
```

*   `port`: Web server listening port.
*   `log_dir`: Log storage directory.
*   `log_retention_days`: Log retention days.
*   `broadcast_ip`: Broadcast address. **Leave empty ("") to automatically scan all interfaces**.
