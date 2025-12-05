# WOL Manager

[中文](README.md) | [English](README_EN.md)

这是一个简单而强大的 Wake-on-LAN (WOL) 网络唤醒管理工具。

### 功能特性

*   **智能广播 (Smart Broadcast)**: 
    *   支持指定广播 IP。
    *   **自动发现**: 如果留空广播 IP，系统将自动遍历所有 IPv4 网络接口发送 Magic Packet，确保在多网卡环境下也能成功唤醒。
*   **日志系统**: 
    *   详细记录唤醒操作、设备增删改及系统错误。
    *   支持按设备筛选查看实时日志。
    *   自动日志轮转与清理（默认保留 3 天）。
*   **设备管理**: 轻松添加、编辑和删除需要唤醒的设备，支持设备分组管理。
*   **一键唤醒**: 点击按钮即可发送 Magic Packet 唤醒设备（默认连续发送 5 次以确保成功率）。
*   **状态监测**: 自动通过 ICMP Ping 检测设备在线状态（🟢 在线 / 🔴 离线）。
*   **配置持久化**: 所有配置（包括端口、设备列表、日志设置）存储在 `wol.json` 文件中，方便迁移和备份。
*   **跨平台支持**: 
    *   **Windows**: 支持最小化到系统托盘，提供快捷菜单（打开网页、开机自启、退出）。
    *   **Linux**: 
        *   支持 `-d` 参数以守护进程模式运行。
        *   支持 `-k` 参数停止正在运行的守护进程。
*   **CI/CD**: 集成 GitHub Actions 工作流，支持代码推送自动构建。

### 下载与安装

#### 1. 获取源码

```bash
git clone https://github.com/MajoSissi/wol.git
cd wol
```

#### 2. 安装依赖

本项目依赖 Go 语言环境。

*   **Go 环境**: 请确保安装了 [Go 1.23+](https://go.dev/dl/)。
*   **Linux 额外依赖**: 编译系统托盘库需要 GTK 开发库。
    *   Ubuntu/Debian: `sudo apt-get install libgtk-3-dev libappindicator3-dev`
    *   CentOS/RHEL: `sudo yum install gtk3-devel libappindicator-gtk3-devel`

#### 3. 构建项目

在项目根目录下执行以下命令，构建产物将生成在 `build/` 目录中：

**Windows (PowerShell)**
```powershell
# 创建 build 目录
mkdir build

# 构建 Windows 版本 (无控制台窗口)
$env:GOOS='windows'; $env:GOARCH='amd64'; go build -ldflags "-H=windowsgui" -o build/wol-windows.exe
```

**Linux (Bash)**
```bash
# 创建 build 目录
mkdir -p build

# 构建 Linux 版本
GOOS=linux GOARCH=amd64 go build -o build/wol-linux
```

### 运行指南

#### Windows
1.  双击 `build/wol-windows.exe` 运行。
2.  程序启动后会最小化到右下角系统托盘。
3.  **右键点击托盘图标**:
    *   **Open Web Page**: 在浏览器中打开管理界面 (默认 http://localhost:8888)。
    *   **Run at Startup**: 勾选后可设置开机自启。
    *   **Exit**: 退出程序。

#### Linux
*   **前台运行**:
    ```bash
    ./wol
    ```
*   **后台守护进程**:
    ```bash
    ./wol -d
    ```
*   **停止守护进程**:
    ```bash
    ./wol -k
    ```

### 配置文件说明 (`wol.json`)

程序首次运行会自动生成此文件。

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

*   `port`: Web 服务监听端口。
*   `log_dir`: 日志存储目录。
*   `log_retention_days`: 日志保留天数。
*   `broadcast_ip`: 广播地址。**留空 ("") 表示自动扫描所有接口**。
