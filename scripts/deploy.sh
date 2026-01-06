#!/bin/bash
# Cyber Range Deployment Script
# Runs OpenTofu, exports LXD instance list, starts config server

set -e

# Configuration - modify these
PROJECT_NAME="${PROJECT_NAME:-homelab-dcig}"
SERVER_PORT="${SERVER_PORT:-8080}"
INSTANCES_FILE="${INSTANCES_FILE:-instances.json}"
IDLE_TIMEOUT="${IDLE_TIMEOUT:-15m}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Run OpenTofu
run_tofu() {
    log_info "Running OpenTofu apply..."
    tofu apply
    log_info "OpenTofu apply completed"
}

# Wait for VMs to be ready
wait_for_vms() {
    log_info "Waiting for VMs to initialize..."
    sleep 10
}

# Export LXD instance list
export_instances() {
    log_info "Switching to project: $PROJECT_NAME"
    lxc project switch "$PROJECT_NAME"
    
    log_info "Exporting instance list to $INSTANCES_FILE..."
    lxc list --format json > "$INSTANCES_FILE"
    
    instance_count=$(jq length "$INSTANCES_FILE" 2>/dev/null || echo "?")
    log_info "Exported $instance_count instances"
}

# Start the config server
start_server() {
    log_info "Starting configuration server on port $SERVER_PORT..."
    
    # Kill any existing server
    pkill -f "./server" 2>/dev/null || true
    
    # Start server in background with idle timeout
    nohup ./server -listen ":$SERVER_PORT" -instances "$INSTANCES_FILE" -idle-timeout "$IDLE_TIMEOUT" > server.log 2>&1 &
    SERVER_PID=$!
    
    sleep 2
    
    if ps -p $SERVER_PID > /dev/null; then
        log_info "Server started (PID: $SERVER_PID)"
        log_info "Server will auto-shutdown after $IDLE_TIMEOUT of inactivity"
        log_info "Server log: server.log"
    else
        log_error "Server failed to start. Check server.log"
        exit 1
    fi
}

# Main
main() {
    echo "=========================================="
    echo "  Cyber Range Deployment"
    echo "  Project: $PROJECT_NAME"
    echo "=========================================="
    echo ""
    
    run_tofu
    wait_for_vms
    export_instances
    start_server
    
    echo ""
    log_info "Deployment complete!"
    echo ""
    echo "Server running at: http://$(hostname -I | awk '{print $1}'):$SERVER_PORT"
    echo "Idle timeout: $IDLE_TIMEOUT (server will auto-shutdown after no requests)"
    echo ""
    echo "Endpoints:"
    echo "  GET  /config?mac=XX:XX:XX:XX:XX:XX  - Get VM config"
    echo "  POST /reload                         - Reload instances.json"
    echo "  GET  /status                         - Check server status"
    echo ""
    echo "Windows VMs will automatically configure themselves on boot."
    echo "Server will shutdown automatically after $IDLE_TIMEOUT of inactivity."
}

# Run main
main "$@"
