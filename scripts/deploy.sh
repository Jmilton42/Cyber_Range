#!/bin/bash
# Cyber Range Deployment Script
# Runs OpenTofu, exports LXD instance list, starts config server

set -e

# Configuration - modify these
PROJECT_NAME="${PROJECT_NAME:-dcig}"
SERVER_PORT="${SERVER_PORT:-8080}"
INSTANCES_FILE="${INSTANCES_FILE:-instances.json}"
IDLE_TIMEOUT="${IDLE_TIMEOUT:-5m}"
SUBNETS_FILE="${SUBNETS_FILE:-subnets.json}"

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

init_subnets_file() {
    if [ ! -f "$SUBNETS_FILE" ]; then
        log_info "Creating subnets file: $SUBNETS_FILE"
        echo '{"allocations":[]}' > "$SUBNETS_FILE"
    fi
}

# Get next available subnet octet
get_next_subnet() {
    init_subnets_file
    
    # Check if project already has an allocation
    existing=$(jq -r --arg proj "$PROJECT_NAME" '.allocations[] | select(.project == $proj) | .subnet_octet' "$SUBNETS_FILE")
    
    if [ -n "$existing" ]; then
        log_info "Project '$PROJECT_NAME' already has subnet octet: $existing"
        echo "$existing"
        return
    fi
    
    # Find next available octet (starting from 1)
    used_octets=$(jq -r '.allocations[].subnet_octet' "$SUBNETS_FILE" | sort -n)
    
    next_octet=1
    for used in $used_octets; do
        if [ "$next_octet" -eq "$used" ]; then
            next_octet=$((next_octet + 1))
        else
            break
        fi
    done
    
    # Cap at 254 (valid third octet range)
    if [ "$next_octet" -gt 254 ]; then
        log_error "No available subnet octets (all 1-254 are allocated)"
        exit 1
    fi
    
    echo "$next_octet"
}

# Allocate subnet for project
allocate_subnet() {
    local octet=$1
    init_subnets_file
    
    # Check if already allocated
    existing=$(jq -r --arg proj "$PROJECT_NAME" '.allocations[] | select(.project == $proj) | .subnet_octet' "$SUBNETS_FILE")
    
    if [ -n "$existing" ]; then
        log_info "Subnet already allocated for project '$PROJECT_NAME'"
        return
    fi
    
    log_info "Allocating subnet octet $octet for project '$PROJECT_NAME'"
    
    # Add allocation
    timestamp=$(date -Iseconds)
    jq --arg proj "$PROJECT_NAME" \
       --argjson octet "$octet" \
       --arg ts "$timestamp" \
       '.allocations += [{"project": $proj, "subnet_octet": $octet, "allocated_at": $ts}]' \
       "$SUBNETS_FILE" > "${SUBNETS_FILE}.tmp" && mv "${SUBNETS_FILE}.tmp" "$SUBNETS_FILE"
    
    log_info "Subnet allocated: 10.0.${octet}.0/24"
}

# Run OpenTofu
run_tofu() {
    local subnet_octet=$1
    log_info "Running OpenTofu apply with guac_subnet_octet=$subnet_octet..."
    tofu apply -var "guac_subnet_octet=$subnet_octet" -var "project_name=$PROJECT_NAME"
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
    pgrep -x server | xargs -r kill 2>/dev/null || true
    
    # Start server in background with idle timeout
    nohup ./server -listen "10.0.14.6:$SERVER_PORT" -instances "$INSTANCES_FILE" -idle-timeout "$IDLE_TIMEOUT" > server.log 2>&1 &
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

start_win() {
    log_info "Starting Windows VMs..."
    bash /home/ceroc/InSPIRE/bin/scripts/start_win.sh "$PROJECT_NAME"
}

# Main
main() {
    echo "=========================================="
    echo "  Cyber Range Deployment"
    echo "  Project: $PROJECT_NAME"
    echo "=========================================="
    echo ""

    SUBNET_OCTET=$(get_next_subnet)
    allocate_subnet "$SUBNET_OCTET"
    
    log_info "Using guac subnet: 10.0.${SUBNET_OCTET}.0/24"
    echo ""
    
    run_tofu "$SUBNET_OCTET"
    wait_for_vms
    export_instances
    start_server
    start_win
    
    echo ""
    log_info "Deployment complete!"
    echo ""
    echo "Server running at: http://10.0.14.6:$SERVER_PORT"
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
