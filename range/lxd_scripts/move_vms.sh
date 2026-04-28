#!/bin/bash

PROJECT_NAME="$1"
TARGET_NODE="$2"
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Get instance names in the specified project
INSTANCES=$(lxc list --project "$PROJECT_NAME" --format json | jq -r '.[].name')

# swap to project
lxc project switch "$PROJECT_NAME"
# Loop through each instance and create a snapshot
for INSTANCE in $INSTANCES; do
	echo -e "${NC}Migrating instance '$INSTANCE'"
	lxc move "$INSTANCE" --target "$TARGET_NODE"
	sleep 10
done

echo -e "${NC}Migrate process completed for '$PROJECT_NAME'."
