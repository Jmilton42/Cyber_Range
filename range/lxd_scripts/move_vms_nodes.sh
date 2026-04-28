#!/bin/bash
#
# Migrate every instance in a project that is currently running on
# SOURCE_NODE over to TARGET_NODE.
#
# Usage: move_vms_nodes.sh <project> <target-node> <source-node>
# Example: move_vms_nodes.sh CIG-Lab micro-01 micro-05

PROJECT_NAME="$1"
TARGET_NODE="$2"
SOURCE_NODE="$3"
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

if [[ -z "$PROJECT_NAME" || -z "$TARGET_NODE" || -z "$SOURCE_NODE" ]]; then
	echo "Usage: $0 <project> <target-node> <source-node>" >&2
	echo "Example: $0 CIG-Lab micro-01 micro-05" >&2
	exit 1
fi

# Get instance names in the project that currently live on SOURCE_NODE
INSTANCES=$(lxc list --project "$PROJECT_NAME" --format json | jq -r --arg src "$SOURCE_NODE" '.[] | select(.location == $src) | .name')

if [[ -z "$INSTANCES" ]]; then
	echo -e "${NC}No instances found in project '$PROJECT_NAME' on node '$SOURCE_NODE'. Nothing to do."
	exit 0
fi

# swap to project
lxc project switch "$PROJECT_NAME"

# Loop through each instance on the source node and migrate it to the target
for INSTANCE in $INSTANCES; do
	echo -e "${NC}Stopping '$INSTANCE' on node '$SOURCE_NODE'..."
	lxc stop "$INSTANCE" --force
	echo -e "${NC}Migrating '$INSTANCE' from '$SOURCE_NODE' to '$TARGET_NODE'..."
	lxc move "$INSTANCE" --target "$TARGET_NODE"
	echo -e "${NC}Starting '$INSTANCE' on node '$TARGET_NODE'..."
	lxc start "$INSTANCE"
	sleep 10
done

echo -e "${NC}Migrate process completed for '$PROJECT_NAME'."