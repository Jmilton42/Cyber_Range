#!/usr/bin/env bash
# Build forge plus all bundled forge-* plugins.
#
# Usage:
#   forge/scripts/build_all.sh                    # build into ./bin/
#   OUT=/usr/local/bin sudo forge/scripts/build_all.sh
#   INSTALL=1 forge/scripts/build_all.sh          # also copy into /home/ceroc/InSPIRE/bin/forge_bin
#
# Exits non-zero if any binary fails to build.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUT="${OUT:-${ROOT_DIR}/bin}"
INSTALL_DIR="${INSTALL_DIR:-/home/ceroc/InSPIRE/bin/forge_bin}"

mkdir -p "${OUT}"

cd "${ROOT_DIR}"

build() {
    local name="$1"
    local pkg="$2"
    echo "[BUILD] ${name} -> ${OUT}/${name}"
    go build -o "${OUT}/${name}" "${pkg}"
}

build forge                ./cmd/forge
build forge-snapshot       ./cmd/forge-snapshot
build forge-start          ./cmd/forge-start
build forge-stop           ./cmd/forge-stop
build forge-migrate        ./cmd/forge-migrate
build forge-networks-prune ./cmd/forge-networks-prune
build forge-cost           ./cmd/forge-cost

if [ "${INSTALL:-}" = "1" ]; then
    echo "[INSTALL] Copying binaries into ${INSTALL_DIR}"
    install -d "${INSTALL_DIR}"
    install -m 0755 "${OUT}"/forge "${OUT}"/forge-snapshot "${OUT}"/forge-start \
                   "${OUT}"/forge-stop "${OUT}"/forge-migrate "${OUT}"/forge-networks-prune \
                   "${OUT}"/forge-cost \
                   "${INSTALL_DIR}/"
fi

echo "[OK] Built all forge binaries into ${OUT}"
