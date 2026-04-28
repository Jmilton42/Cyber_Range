#!/bin/bash

PROJECT_NAME="$1"
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Get instance names in the specified project
INSTANCES=$(lxc list --project "$PROJECT_NAME" --format json | jq -r '.[].name')

# swap to project
lxc project switch "$PROJECT_NAME"
# Loop through each instance and create a snapshot
for INSTANCE in $INSTANCES; do
  echo -e "${NC}Starting instance '$INSTANCE' in project '$PROJECT_NAME'..."

  lxc start "$INSTANCE"
  if [ $? -eq 0 ]; then
    echo -e  "${GREEN}Successfully started '$INSTANCE'."
  else
    echo -e "${RED}Failed to start '$INSTANCE'. It could already be running."
  fi
  sleep 15
done

echo -e "${NC}Startup process completed for '$PROJECT_NAME'."
