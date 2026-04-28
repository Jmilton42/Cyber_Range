#!/bin/bash
set -euo pipefail

usage() {
  cat <<'EOF'
Delete LXD networks by name prefix.

Usage:
  delete-lxd-networks.sh <prefix> [--project NAME] [--yes] [--dry-run]

Options:
  <prefix>         Required. Networks whose names start with this prefix will be targeted.
  --project NAME   (Optional) LXD project to operate in. Defaults to the current project.
  --yes            Skip interactive confirmation.
  --dry-run        Show what would be deleted without deleting anything.

Examples:
  delete-lxd-networks.sh teamA-
  delete-lxd-networks.sh csc- --project CSC-3570 --dry-run
  delete-lxd-networks.sh testnet- --yes
EOF
}

# --- parse args ---
if [[ ${1:-} == "-h" || ${1:-} == "--help" || ${#} -lt 1 ]]; then
  usage; exit 0
fi

PREFIX="$1"; shift || true
PROJECT=""
ASSUME_YES="false"
DRY_RUN="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --project)
      PROJECT="$2"; shift 2;;
    --yes)
      ASSUME_YES="true"; shift;;
    --dry-run)
      DRY_RUN="true"; shift;;
    *)
      echo "Unknown option: $1" >&2
      usage; exit 1;;
  esac
done

# --- build base lxc command with optional project ---
LXC_CMD=(lxc)
if [[ -n "$PROJECT" ]]; then
  LXC_CMD+=(--project "$PROJECT")
fi

# --- fetch network names (CSV column 'n' = name) ---
# Works on recent LXD: 'lxc network list -c n -f csv'
mapfile -t ALL_NETS < <("${LXC_CMD[@]}" network list -f csv | cut -d, -f1 ) #sed '/^\s*$/d')

# --- filter by prefix safely (no regex needed) ---
MATCHES=()
for n in "${ALL_NETS[@]}"; do
  # strip possible whitespace
  n="${n#"${n%%[![:space:]]*}"}"; n="${n%"${n##*[![:space:]]}"}"
  [[ -z "$n" ]] && continue
  if [[ "$n" == "$PREFIX"* ]]; then
    MATCHES+=("$n")
  fi
done

if [[ ${#MATCHES[@]} -eq 0 ]]; then
  echo "No networks found starting with prefix: '$PREFIX'${PROJECT:+ in project '$PROJECT'}"
  exit 0
fi

echo "Found ${#MATCHES[@]} network(s) matching prefix '${PREFIX}'${PROJECT:+ in project '$PROJECT'}:"
for n in "${MATCHES[@]}"; do
  echo "  - $n"
done

if [[ "$DRY_RUN" == "true" ]]; then
  echo
  echo "[DRY-RUN] No changes made."
  exit 0
fi

if [[ "$ASSUME_YES" != "true" ]]; then
  echo
  read -r -p "Type the exact prefix '${PREFIX}' to confirm deletion of these networks: " CONFIRM
  if [[ "$CONFIRM" != "$PREFIX" ]]; then
    echo "Confirmation did not match. Aborting."
    exit 1
  fi
fi

echo
echo "Deleting networks..."
for n in "${MATCHES[@]}"; do
  echo "Deleting: $n"
  # If a network is in use (profiles/instances), this may fail; handle gracefully.
  if ! "${LXC_CMD[@]}" network delete "$n"; then
    echo "WARNING: Failed to delete '$n'. It may be in use by instances or profiles." >&2
  fi
done

echo "Done."
