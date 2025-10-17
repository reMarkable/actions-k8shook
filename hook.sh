#!/usr/bin/env bash

echo
echo "=== Environment variables ==="
env | egrep "(GITHUB|ACTIONS|HOOK)" | sort
if [ -n "$GITHUB_ENV" ]; then
  echo "=== GITHUB_ENV ==="
  cat "$GITHUB_ENV"
fi
if [ -n "$GITHUB_STATE" ]; then
  echo "=== GITHUB_STATE ==="
  cat "$GITHUB_STATE"
fi
if [ -n "$GITHUB_STEP_SUMMARY" ]; then
  echo "=== GITHUB_STEP_SUMMARY ==="
  cat "$GITHUB_STEP_SUMMARY"
fi
/home/marcus/temp/hook/hook
