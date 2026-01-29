# Cyber Range Configuration Agent - Setup Guide

Automatically configures Windows, Linux, and OpenWrt VM/container hostname and network settings based on LXD instance configuration.

## How It Works

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  deploy.sh   │     │  Go Server   │     │  Win/Linux/  │
│              │     │              │     │   OpenWrt    │
│ 1. tofu apply│     │              │     │              │
│ 2. lxc list  │────▶│ 3. Load JSON │     │              │
│    --json    │     │              │     │ 4. Boot      │
│              │     │              │◀────│ 5. Request   │
│              │     │ 6. Lookup    │     │    config    │
│              │     │    by MAC    │────▶│ 7. Apply     │
│              │     │              │     │    settings  │
└──────────────┘     └──────────────┘     └──────────────┘
```

1. `deploy.sh` runs OpenTofu to create VMs
2. Script exports `lxc list --format json` to `instances.json`
3. Go server loads the JSON file
4. VMs/containers boot (with 0-30 second random delay)
5. Client requests config using its MAC address
6. Server finds matching instance, returns hostname + network config
7. Client applies settings and creates marker file (won't run again)
8. **Server auto-shuts down after 15 minutes of inactivity**

---

## Quick Start

### 1. Build Binaries

```powershell
cd Cyber_Range

# Download dependencies
go mod tidy

# Build server (Linux)
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o server ./cmd/server

# Build Windows client
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o client.exe ./cmd/client/windows

# Build Linux client
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o linux-client ./cmd/client/linux

# Build OpenWrt client (Linux x86_64)
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o openwrt-client ./cmd/client/openwrt
```

### 2. Prepare Windows Base Image

On your `windows-10-base` VM:

```powershell
# Allows scripts to run
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force

# Create directory
New-Item -Path "C:\ProgramData\cyber-range" -ItemType Directory -Force

# Copy client.exe to C:\ProgramData\cyber-range\

# Create scheduled task (as Administrator)
.\setup-task.ps1 -ServerURL "http://YOUR_SERVER_IP:8080"
```

Then snapshot/export the image.

### 3. Prepare OpenWrt Base Image

On your OpenWrt container/VM:

```bash
# Create directory
mkdir -p /etc/cyber-range

# Copy openwrt-client binary
scp openwrt-client root@openwrt:/etc/cyber-range/

# Make executable
chmod +x /etc/cyber-range/openwrt-client

# Create startup script
cat > /etc/init.d/cyber-range-config << 'EOF'
#!/bin/sh /etc/rc.common
START=99
STOP=10

start() {
    /etc/cyber-range/openwrt-client -server "http://YOUR_SERVER_IP:8080" -interface eth1
}
EOF

chmod +x /etc/init.d/cyber-range-config
/etc/init.d/cyber-range-config enable
```

Then snapshot/export the image.

### 4. Prepare Linux Base Image

On your Linux VM (Ubuntu, Debian, RHEL, CentOS, Fedora, etc.):

```bash
# Create directory
sudo mkdir -p /var/lib/cyber-range

# Copy linux-client binary
sudo cp linux-client /var/lib/cyber-range/
sudo chmod +x /var/lib/cyber-range/linux-client

# Create systemd service
sudo tee /etc/systemd/system/cyber-range-config.service << 'EOF'
[Unit]
Description=Cyber Range Configuration Client
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/var/lib/cyber-range/linux-client -server "http://YOUR_SERVER_IP:8080"
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

# Enable service
sudo systemctl daemon-reload
sudo systemctl enable cyber-range-config.service
```

Then snapshot/export the image.

### 5. Deploy

Copy to your OpenTofu box:
- `server` binary
- `scripts/deploy.sh`
- Your Terraform files

Run:

```bash
chmod +x deploy.sh server
PROJECT_NAME="homelab-dcig" ./deploy.sh
```

---

## Detailed Setup

### Server Setup (OpenTofu Box)

The server reads from `instances.json` (exported by `lxc list --format json`).

**Option A: Use deploy.sh (recommended)**

```bash
PROJECT_NAME="homelab-dcig" ./deploy.sh
```

This will:
1. Run `tofu apply`
2. Export instances to JSON
3. Start the server (auto-shuts down after 15 min idle)

**Environment variables:**
| Variable | Default | Description |
|----------|---------|-------------|
| `PROJECT_NAME` | `homelab-dcig` | LXD project name |
| `SERVER_PORT` | `8080` | Server listen port |
| `IDLE_TIMEOUT` | `15m` | Auto-shutdown timeout |

**Option B: Manual**

```bash
# Run tofu
tofu apply

# Export instances
lxc project switch homelab-dcig
lxc list --format json > instances.json

# Start server (auto-shutdown after 15 min idle)
./server -instances instances.json -listen :8080 -idle-timeout 15m
```

### Windows Client Setup

1. Copy `client.exe` to `C:\ProgramData\cyber-range\`

2. Run setup script as Administrator:
   ```powershell
   .\setup-task.ps1 -ServerURL "http://SERVER_IP:8080"
   ```

3. Snapshot the VM as your new base image

**What it does:**
- Sets hostname from LXD instance name
- Configures network (static IP or DHCP)
- Reboots to apply hostname change

### OpenWrt Client Setup

1. Copy `openwrt-client` to `/etc/cyber-range/`

2. Create init script:
   ```bash
   cat > /etc/init.d/cyber-range-config << 'EOF'
   #!/bin/sh /etc/rc.common
   START=99
   STOP=10

   start() {
       /etc/cyber-range/openwrt-client -server "http://SERVER_IP:8080" -interface eth1
   }
   EOF
   
   chmod +x /etc/init.d/cyber-range-config
   /etc/init.d/cyber-range-config enable
   ```

3. Snapshot the container as your new base image

**What it does:**
- Configures network interfaces via UCI (does NOT change hostname)
- Maps cloud-init interface names to UCI names:
  - `eth0`, `eth-0` → `wan`
  - `eth1`, `eth-1` → `lan`
  - `eth2`, `eth-2` → `lan2`
- Restarts network service to apply changes

### Linux Client Setup

1. Copy `linux-client` to `/var/lib/cyber-range/`

2. Create systemd service:
   ```bash
   sudo tee /etc/systemd/system/cyber-range-config.service << 'EOF'
   [Unit]
   Description=Cyber Range Configuration Client
   After=network-online.target
   Wants=network-online.target

   [Service]
   Type=oneshot
   ExecStart=/var/lib/cyber-range/linux-client -server "http://SERVER_IP:8080"
   RemainAfterExit=yes

   [Install]
   WantedBy=multi-user.target
   EOF
   
   sudo systemctl daemon-reload
   sudo systemctl enable cyber-range-config.service
   ```

3. Snapshot the VM as your new base image

**What it does:**
- Sets hostname via `hostnamectl` (works on all systemd-based distros)
- Auto-detects and configures network using:
  1. NetworkManager (`nmcli`) - if available and running
  2. Netplan - if `/etc/netplan/` exists (Ubuntu 18.04+)
  3. ifupdown - if `/etc/network/interfaces` exists (older Debian)
- Reboots to ensure all changes take effect

**Supported Distros:**
- Ubuntu 16.04+
- Debian 8+
- RHEL/CentOS 7+
- Fedora
- Any systemd-based distribution

### Terraform Configuration

Add `cloud-init.network-config` to your instances:

**Windows - Single Interface (Static IP):**
```hcl
config = {
  "cloud-init.network-config" = <<-EOF
    version: 2
    ethernets:
      eth-0:
        dhcp4: false
        addresses:
          - 192.168.1.100/24
        routes:
          - to: default
            via: 192.168.1.1
        nameservers:
          addresses: [192.168.1.1]
    EOF
}
```

**Windows - DHCP:**
```hcl
config = {
  "cloud-init.network-config" = "DHCP"
}
```

**OpenWrt - Multiple Interfaces:**
```hcl
config = {
  "cloud-init.network-config" = <<-EOF
    version: 2
    ethernets:
      eth0:
        dhcp4: true
      eth1:
        dhcp4: false
        addresses:
          - 192.168.1.1/24
      eth2:
        dhcp4: false
        addresses:
          - 172.31.31.1/24
    EOF
}
```

---

## Configuration Reference

### Server

**Command line:**
```bash
./server -instances instances.json -listen :8080 -idle-timeout 15m
```

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `-instances` | `instances.json` | Path to LXD instances JSON file |
| `-listen` | `:8080` | Listen address |
| `-idle-timeout` | `15m` | Auto-shutdown after inactivity (0 to disable) |
| `-config` | `config.yaml` | Path to config file |

**Config file (config.yaml):**
```yaml
listen: ":8080"
instances_file: "./instances.json"
```

**Endpoints:**
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/config?mac=XX:XX:XX:XX:XX:XX` | GET | Get config for MAC address |
| `/reload` | POST | Reload instances.json |
| `/status` | GET | Check server status and idle time |

**API Response:**
```json
{
  "hostname": "team1-win10",
  "network": { "dhcp": false, "address": "192.168.1.15/24", "gateway": "192.168.1.1" },
  "networks": {
    "eth-0": { "dhcp": false, "address": "192.168.1.15/24", "gateway": "192.168.1.1" },
    "eth-1": { "dhcp": true }
  }
}
```

### Windows Client

**Command line:**
```bash
client.exe -server http://server:8080
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-server` | Server URL (required) |
| `-interface` | Specific network interface name |
| `-no-delay` | Skip random startup delay |

**Files:**
| Path | Description |
|------|-------------|
| `C:\ProgramData\cyber-range\client.exe` | Client binary |
| `C:\ProgramData\cyber-range\.configured` | Marker file (prevents re-run) |
| `C:\ProgramData\cyber-range\config.log` | Log file |

### OpenWrt Client

**Command line:**
```bash
./openwrt-client -server http://server:8080 -interface eth1
```

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `-server` | (required) | Server URL |
| `-interface` | `eth1` | Interface for MAC lookup |
| `-no-delay` | false | Skip random startup delay |

**Files:**
| Path | Description |
|------|-------------|
| `/etc/cyber-range/openwrt-client` | Client binary |
| `/etc/cyber-range/.configured` | Marker file (prevents re-run) |
| `/etc/cyber-range/config.log` | Log file |

**Interface Mapping:**
| Cloud-init | UCI Interface |
|------------|---------------|
| `eth0`, `eth-0` | `wan` |
| `eth1`, `eth-1` | `lan` |
| `eth2`, `eth-2` | `lan2` |
| `eth3`, `eth-3` | `lan3` |

### Linux Client

**Command line:**
```bash
./linux-client -server http://server:8080
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-server` | Server URL (required) |
| `-interface` | Specific network interface name |
| `-no-delay` | Skip random startup delay |

**Files:**
| Path | Description |
|------|-------------|
| `/var/lib/cyber-range/linux-client` | Client binary |
| `/var/lib/cyber-range/.configured` | Marker file (prevents re-run) |
| `/var/lib/cyber-range/config.log` | Log file |

**Network Configuration Methods (auto-detected):**
| Method | Detection | Distros |
|--------|-----------|---------|
| NetworkManager | `nmcli` exists and service running | Most modern distros |
| Netplan | `/etc/netplan/` exists | Ubuntu 18.04+ |
| ifupdown | `/etc/network/interfaces` exists | Older Debian/Ubuntu |

**Retry Behavior:**
- Linux and OpenWrt clients retry for up to **60 minutes** (60 retries × 60 seconds)
- This allows time for long infrastructure builds to complete
- Windows client uses a shorter retry window

---

## Troubleshooting

### Check Client Logs

**Windows:**
```
C:\ProgramData\cyber-range\config.log
```

**Linux:**
```bash
cat /var/lib/cyber-range/config.log
```

**OpenWrt:**
```bash
cat /etc/cyber-range/config.log
```

### Check Server Logs

```bash
tail -f server.log
```

### Test Server Manually

```bash
curl "http://localhost:8080/config?mac=00:16:3e:4f:e5:74"
```

### Reload Instances

If you add new VMs:

```bash
lxc list --format json > instances.json
curl -X POST http://localhost:8080/reload
```

### Reset a Windows VM

```powershell
Remove-Item "C:\ProgramData\cyber-range\.configured" -Force
Remove-Item "C:\ProgramData\cyber-range\config.log" -Force
Restart-Computer
```

### Reset a Linux VM

```bash
sudo rm /var/lib/cyber-range/.configured
sudo rm /var/lib/cyber-range/config.log
sudo systemctl restart cyber-range-config.service
# Or reboot:
sudo reboot
```

### Reset an OpenWrt Container

```bash
rm /etc/cyber-range/.configured
rm /etc/cyber-range/config.log
/etc/init.d/cyber-range-config start
```

### Common Issues

| Issue | Solution |
|-------|----------|
| Client can't reach server | Check firewall, verify server IP |
| "Instance not found" | VM might not be in instances.json - run `/reload` |
| Hostname not changed | Requires reboot (Windows only) |
| "Already configured" | Delete `.configured` marker file |
| OpenWrt interface wrong | Check interface mapping or use `-interface` flag |

---

## File Structure

```
Cyber_Range/
├── go.mod
├── cmd/
│   ├── server/main.go           # Server entry point
│   └── client/
│       ├── windows/main.go      # Windows client
│       ├── linux/main.go        # Linux client
│       └── openwrt/main.go      # OpenWrt client
├── internal/
│   ├── config/types.go          # Shared types
│   ├── server/server.go         # Server logic
│   └── client/
│       ├── common/
│       │   └── mac.go           # MAC address (shared)
│       ├── windows/
│       │   ├── hostname.go      # Hostname change
│       │   ├── network.go       # Network config (netsh)
│       │   ├── reboot.go        # System reboot
│       │   └── marker.go        # Run-once marker
│       ├── linux/
│       │   ├── hostname.go      # Hostname change (hostnamectl)
│       │   ├── network.go       # Network config (auto-detect)
│       │   ├── reboot.go        # System reboot
│       │   └── marker.go        # Run-once marker
│       └── openwrt/
│           ├── network.go       # Network config (UCI)
│           ├── restart.go       # Network restart
│           └── marker.go        # Run-once marker
├── scripts/
│   ├── deploy.sh                # Deployment script
│   └── setup-task.ps1           # Windows task setup
├── config.yaml.example
└── SETUP.md
```
