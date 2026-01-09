# Cyber Range Configuration Agent - Setup Guide

Automatically configures Windows VM hostname and network settings based on LXD instance configuration.

## How It Works

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  deploy.sh   │     │  Go Server   │     │  Windows VM  │
│              │     │              │     │              │
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
4. Windows VMs boot (with 0-30 second random delay)
5. Client requests config using its MAC address
6. Server finds matching instance, returns hostname + network config
7. Client applies settings and creates marker file (won't run again)
8. **Server auto-shuts down after 15 minutes of inactivity**

---

## Quick Start

### 1. Build Binaries

```bash
cd Cyber_Range

# Download dependencies
go mod tidy

# Build server (Linux)
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o server ./cmd/server

# Build client (Windows)
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o client.exe ./cmd/client
```

### 2. Prepare Windows Base Image

On your `windows-10-base` VM:

```powershell
# Create directory
New-Item -Path "C:\ProgramData\cyber-range" -ItemType Directory -Force

# Copy client.exe to C:\ProgramData\cyber-range\

# Create scheduled task (as Administrator)
.\setup-task.ps1 -ServerURL "http://YOUR_SERVER_IP:8080"
```

Then snapshot/export the image.

### 3. Deploy

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

### Client Setup (Windows Image)

1. Copy `client.exe` to `C:\ProgramData\cyber-range\`

2. Run setup script as Administrator:
   ```powershell
   .\setup-task.ps1 -ServerURL "http://SERVER_IP:8080"
   ```

3. Snapshot the VM as your new base image

### Terraform Configuration

Add `cloud-init.network-config` to your Windows instances:

**DHCP:**
```hcl
config = {
  "cloud-init.network-config" = "DHCP"
}
```

**Static IP:**
```hcl
config = {
  "cloud-init.network-config" = <<-EOF
    network:
      version: 2
      ethernets:
        eth-1:
          addresses:
            - 192.168.1.100/24
          routes:
            - to: default
              via: 192.168.1.1
          nameservers:
            addresses:
              - 8.8.8.8
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

**Auto-Shutdown:**
The server automatically shuts down after 15 minutes of no requests. This ensures the server doesn't run indefinitely after all VMs are configured.

### Client

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

---

## Troubleshooting

### Check Client Logs

```
C:\ProgramData\cyber-range\config.log
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

### Common Issues

| Issue | Solution |
|-------|----------|
| Client can't reach server | Check firewall, verify server IP |
| "Instance not found" | VM might not be in instances.json - run `/reload` |
| Hostname not changed | Requires reboot |
| "Already configured" | Delete `.configured` marker file |

---

## File Structure

```
Cyber_Range/
├── go.mod
├── cmd/
│   ├── server/main.go      # Server entry point
│   └── client/main.go      # Client entry point
├── internal/
│   ├── config/types.go     # Shared types
│   ├── server/server.go    # Server logic
│   └── client/
│       ├── mac.go          # MAC address
│       ├── hostname.go     # Hostname change
│       ├── network.go      # Network config
│       └── marker.go       # Run-once marker
├── scripts/
│   ├── deploy.sh           # Deployment script
│   └── setup-task.ps1      # Windows task setup
├── config.yaml.example
└── SETUP.md
```
