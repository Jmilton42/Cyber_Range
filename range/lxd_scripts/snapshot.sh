#!/bin/bash

PROJECT_NAME="$1"
SNAPSHOT_NAME="daily-snapshot-$(date +%Y%m%d%H%M)" # Unique name with timestamp
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Get instance names in the specified project
INSTANCES=$(lxc list --project "$PROJECT_NAME" --format json | jq -r '.[].name')

# Loop through each instance and create a snapshot
for INSTANCE in $INSTANCES; do
  echo -e "${NC}Creating snapshot '$SNAPSHOT_NAME' for instance '$INSTANCE' in project '$PROJECT_NAME'..."
  lxc snapshot "$INSTANCE" "$SNAPSHOT_NAME" --project "$PROJECT_NAME"
  if [ $? -eq 0 ]; then
    echo -e  "${GREEN}Successfully created snapshot for '$INSTANCE'."
  else
    echo -e "${RED}Failed to create snapshot for '$INSTANCE'."
  fi
  sleep 15
done

echo -e "${NC}Snapshot process complete for project '$PROJECT_NAME'."
