#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${SSB_BENCHMARK_ROOT:-}" ]]; then
  echo "error: SSB_BENCHMARK_ROOT must name the directory containing the pinned benchmark clones" >&2
  exit 2
fi

repository_root="$(git rev-parse --show-toplevel)"
benchmark_work="$(mktemp -d "${TMPDIR:-/tmp}/ssb-inventory-benchmark.XXXXXX")"
benchmark_binary="$benchmark_work/inventory.test"

cleanup() {
  rm -f "$benchmark_binary" "$benchmark_work/benchmark-output.txt"
  rmdir "$benchmark_work"
}
trap cleanup EXIT

cd "$repository_root"

go test ./internal/evaluation -run TestPinnedInventoryResourceEnvelope -v
go test -c -o "$benchmark_binary" ./internal/inventory

echo "Cold batch-policy sweep (discarded)"
"$benchmark_binary" \
  -test.run '^$' \
  -test.bench '^BenchmarkBatchPolicies$' \
  -test.benchmem \
  -test.benchtime=1x \
  -test.count=1 >/dev/null

echo "Five warm batch-policy runs"
"$benchmark_binary" \
  -test.run '^$' \
  -test.bench '^BenchmarkBatchPolicies$' \
  -test.benchmem \
  -test.benchtime=1x \
  -test.count=5

for batch_mib in 4 8 16; do
  benchmark_pattern="^BenchmarkBatchPolicies/512_entries_${batch_mib}_mib$"
  echo "Cold RSS run for entries=512 batch_mib=$batch_mib (discarded)"
  "$benchmark_binary" \
    -test.run '^$' \
    -test.bench "$benchmark_pattern" \
    -test.benchtime=1x \
    -test.count=1 >/dev/null

  for run in 1 2 3 4 5; do
    output_file="$benchmark_work/benchmark-output.txt"
    "$benchmark_binary" \
      -test.run '^$' \
      -test.bench "$benchmark_pattern" \
      -test.benchtime=1x \
      -test.count=1 >"$output_file" 2>&1 &
    benchmark_pid=$!
    peak_kib=0

    while kill -0 "$benchmark_pid" 2>/dev/null; do
      current_kib="$(
        ps -axo pid=,ppid=,rss= |
          awk -v parent="$benchmark_pid" \
            '$1 == parent || $2 == parent { total += $3 } END { print total + 0 }'
      )"
      if (( current_kib > peak_kib )); then
        peak_kib=$current_kib
      fi
      sleep 0.01
    done

    if ! wait "$benchmark_pid"; then
      cat "$output_file"
      exit 1
    fi
    cat "$output_file"
    printf 'combined_peak_rss_bytes=%d entries=512 batch_mib=%d run=%d\n' \
      "$((peak_kib * 1024))" "$batch_mib" "$run"
  done
done
