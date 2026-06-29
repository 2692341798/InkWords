#!/usr/bin/env bash
set -euo pipefail

check_total() {
  local profile="$1"
  local minimum="$2"
  local label="$3"
  local actual
  actual=$(go tool cover -func="$profile" | awk '/^total:/ {gsub("%", "", $3); print $3}')
  awk -v actual="$actual" -v minimum="$minimum" -v label="$label" 'BEGIN {
    if ((actual + 0) < (minimum + 0)) {
      printf "%s coverage %.2f%% is below %.2f%%\n", label, actual, minimum > "/dev/stderr"
      exit 1
    }
    printf "%s coverage %.2f%% meets %.2f%%\n", label, actual, minimum
  }'
}

go test ./... -coverprofile=/tmp/inkwords-all.cover
check_total /tmp/inkwords-all.cover 35.0 "backend total"

go test ./services/llm-stream/app/generation -coverprofile=/tmp/inkwords-generation.cover
check_total /tmp/inkwords-generation.cover 50.0 "llm-stream generation"
