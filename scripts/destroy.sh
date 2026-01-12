#!/bin/bash
# Cyber Range Destroy Script
# Runs OpenTofu destroy and releases subnet allocation

set -e

# Configuration - modify these
PROJECT_NAME="${PROJECT_NAME:-homelab-dcig}"
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

# Get subnet octet for project
get_project_subnet() {
    if [ ! -f "$SUBNETS_FILE" ]; then
        log_error "Subnets file not found: $SUBNETS_FILE"
        exit 1
    fi
    
    local octet=$(jq -r --arg proj "$PROJECT_NAME" '.allocations[] | select(.project == $proj) | .subnet_octet' "$SUBNETS_FILE")
    
    if [ -z "$octet" ] || [ "$octet" = "null" ]; then
        log_error "No subnet allocation found for project '$PROJECT_NAME'"
        exit 1
    fi
    
    echo "$octet"
}

# Release subnet allocation
release_subnet() {
    if [ ! -f "$SUBNETS_FILE" ]; then
        log_warn "Subnets file not found, nothing to release"
        return
    fi
    
    local octet=$(jq -r --arg proj "$PROJECT_NAME" '.allocations[] | select(.project == $proj) | .subnet_octet' "$SUBNETS_FILE")
    
    if [ -z "$octet" ] || [ "$octet" = "null" ]; then
        log_warn "No subnet allocation found for project '$PROJECT_NAME'"
        return
    fi
    
    log_info "Releasing subnet octet $octet for project '$PROJECT_NAME'"
    
    # Remove allocation
    jq --arg proj "$PROJECT_NAME" \
       '.allocations = [.allocations[] | select(.project != $proj)]' \
       "$SUBNETS_FILE" > "${SUBNETS_FILE}.tmp" && mv "${SUBNETS_FILE}.tmp" "$SUBNETS_FILE"
    
    log_info "Subnet released: 10.0.${octet}.0/24"
}

# Run OpenTofu destroy
run_tofu_destroy() {
    local subnet_octet=$1
    log_info "Running OpenTofu destroy with guac_subnet_octet=$subnet_octet..."
    tofu destroy -var "guac_subnet_octet=$subnet_octet" -var "project_name=$PROJECT_NAME"
    log_info "OpenTofu destroy completed"
}

# Kill config server if running
stop_server() {
    log_info "Stopping config server if running..."
    pkill -f "./server" 2>/dev/null || true
}

# Main
main() {
    echo "=========================================="
    echo "  Cyber Range Destroy"
    echo "  Project: $PROJECT_NAME"
    echo "=========================================="
    echo ""
    
    # Get subnet for this project
    SUBNET_OCTET=$(get_project_subnet)
    log_info "Found subnet allocation: 10.0.${SUBNET_OCTET}.0/24"
    echo ""
    
    # Stop the server first
    stop_server
    
    # Run tofu destroy
    run_tofu_destroy "$SUBNET_OCTET"
    
    # Release the subnet allocation
    release_subnet
    
    echo ""
    log_info "Destroy complete!"
    log_info "Subnet 10.0.${SUBNET_OCTET}.0/24 has been released and is available for reuse."
}

# Run main
main "$@"
