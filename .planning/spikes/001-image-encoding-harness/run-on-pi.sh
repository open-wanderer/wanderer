#!/bin/sh
# Runs the matrix on a shared home server without risking its other workloads:
# hard memory cap without swap, lowest CPU/IO weight, nice 19, and first in
# line for the OOM killer. Exceeding the cap kills only this cgroup.
set -eu
cd "$(dirname "$0")"
exec systemd-run --user --scope -q \
  -p MemoryMax=600M -p MemorySwapMax=0 -p CPUWeight=1 -p IOWeight=1 \
  env GOMEMLIMIT=500MiB sh -c 'echo 1000 > /proc/self/oom_score_adj && exec nice -n 19 ./imgbench matrix -minavail 1500 -runs 2 -json results.jsonl "$@"' sh "$@"
