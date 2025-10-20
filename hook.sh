#!/usr/bin/env bash

# echo
echo "=== Environment variables ==="
env | egrep "(RUNNER|GITHUB|ACTIONS|HOOK)" | sort
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"$SCRIPT_DIR/hook"
