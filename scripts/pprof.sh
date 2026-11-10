#!/usr/bin/env bash
# Capture pprof profiles from a running API (start it with PPROF_ADDR=localhost:6060).
#
#   scripts/pprof.sh [cpu|heap|allocs|goroutine] [seconds]
#
# Run alongside the k6 load test, e.g. `scripts/pprof.sh cpu 30`.
# Profiles are written to profiles/ and opened in the pprof web UI.
set -euo pipefail

PPROF_ADDR="${PPROF_ADDR:-localhost:6060}"
kind="${1:-cpu}"
seconds="${2:-30}"

mkdir -p profiles
out="profiles/${kind}-$(date +%Y%m%d-%H%M%S).pb.gz"

case "$kind" in
  cpu) path="profile?seconds=${seconds}" ;;
  heap | allocs | goroutine) path="$kind" ;;
  *) echo "usage: $0 [cpu|heap|allocs|goroutine] [seconds]" >&2; exit 1 ;;
esac

curl -fsS -o "$out" "http://${PPROF_ADDR}/debug/pprof/${path}"
echo "saved $out"
go tool pprof -http=:8081 "$out"
