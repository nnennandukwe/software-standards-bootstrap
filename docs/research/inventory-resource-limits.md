# Inventory resource-limit research

Recorded on 2026-07-24 for the inventory-v2 change based on source commit
`4dc8099`. Measurements used the uncommitted inventory-v2 implementation in
this build and the exact repository pins in `testdata/benchmarks.yaml`.

## Question

What limits bound inspection work without silently excluding supported
repositories, and how should those limits be maintained?

## Primary-source findings

- [GitHub recommends a maximum single Git object of 1 MB and enforces
  100 MB](https://docs.github.com/en/repositories/creating-and-managing-repositories/repository-limits).
  Its repository-size guidance concerns compressed `.git` storage and does not
  imply a cumulative inspection-text limit.
- [GitLab's Semgrep integration defaults `--max-target-bytes` to 1,000,000
  bytes](https://gitlab.com/gitlab-org/gitlab/-/tree/master/doc/user/application_security/sast).
  This independently supports an approximately 1 MiB per-file eligibility
  boundary and visible exclusions.
- [`git cat-file --batch` returns object information followed by full
  uncompressed content](https://git-scm.com/docs/git-cat-file). Batch size
  therefore affects both subprocess overhead and peak resident memory.
- [`bytes.Buffer` grows to retain subprocess
  output](https://pkg.go.dev/bytes#Buffer). The current Git wrapper captures a
  complete batch before inventory parsing, so aggregate inventory bytes are not
  a direct peak-memory control.
- [`cloc` documents its own file limit in terms of measured memory
  amplification](https://github.com/AlDanial/cloc#options). This is a useful
  precedent for deriving limits from the actual workload rather than copying a
  number from an unrelated product.

No primary source supports a 25 MiB repository-wide text limit. GitHub's
separate 25 MiB browser-upload behavior is not an inspection contract.

## Supported-corpus coverage

Host:

- macOS 15.7.3, arm64, Apple M4 Pro
- Git 2.39.5
- Go 1.26.5
- one cold run followed by warm measurements

| Repository | Candidate files | Candidate bytes | Indexed files | Indexed bytes | Warm scan |
|---|---:|---:|---:|---:|---:|
| Cobra | 66 | 705,271 | 65 | 631,792 | 0.058 s |
| Flask | 235 | 1,814,782 | 230 | 1,474,850 | 0.075 s |
| Django | 7,001 | 45,506,636 | 5,619 | 36,820,618 | 0.988 s |
| Next.js | 29,073 | 111,110,455 | 28,403 | 88,643,646 | 3.432 s |

The largest workload is 29,073 files and 111,110,455 bytes. Applying the
agreed 120% rule and readable rounding selects 40,000 files and 128 MiB.
All four repositories complete within the 10-second envelope.

## Batch-policy sweep

Each configuration scanned all four repositories once per repetition. Values
below are five warm `ns/op` observations and the median. Allocation bytes are
representative and measure cumulative allocation work, not peak RSS.

| Entries | MiB | Five warm seconds | Median | Approx. B/op |
|---:|---:|---|---:|---:|
| 32 | 1 | 13.895, 13.819, 14.049, 14.210, 14.166 | 14.049 | 755M |
| 128 | 1 | 5.105, 4.911, 4.941, 4.940, 4.927 | 4.940 | 720M |
| 512 | 1 | 3.269, 3.246, 3.240, 3.246, 3.296 | 3.246 | 760M |
| 32 | 4 | 13.518, 13.416, 13.355, 13.747, 13.770 | 13.518 | 770M |
| 128 | 4 | 4.393, 4.454, 4.388, 4.410, 4.401 | 4.401 | 736M |
| 512 | 4 | 2.144, 2.133, 2.137, 2.141, 2.169 | 2.141 | 739M |
| 32 | 8 | 13.487, 13.469, 13.557, 13.722, 13.867 | 13.557 | 768M |
| 128 | 8 | 4.602, 4.420, 4.384, 4.363, 4.399 | 4.399 | 742M |
| 512 | 8 | 2.056, 2.068, 2.073, 2.074, 2.085 | 2.073 | 731M |
| 32 | 16 | 13.435, 13.456, 13.572, 13.574, 14.002 | 13.572 | 768M |
| 128 | 16 | 4.820, 4.377, 4.407, 4.405, 4.388 | 4.405 | 748M |
| 512 | 16 | 2.041, 2.039, 2.041, 2.049, 2.056 | 2.041 | 724M |

The three 512-entry configurations within 10% of the fastest result were
measured separately with `/usr/bin/time -l`:

| MiB | Five real-time seconds | Median RSS | Maximum RSS |
|---:|---|---:|---:|
| 4 | 2.75, 2.70, 2.72, 2.75, 2.70 | 58,753,024 B | 60,948,480 B |
| 8 | 2.66, 2.63, 2.60, 2.65, 2.63 | 65,077,248 B | 71,335,936 B |
| 16 | 2.61, 2.62, 2.64, 2.61, 2.63 | 79,839,232 B | 81,149,952 B |

Ten-millisecond sampling of the 4 MiB/512-entry test process and its direct Git
children produced five combined high-water observations between 94,650,368 and
96,829,440 bytes. This remains below the 256 MiB envelope.

The selected policy is therefore 512 entries and 4 MiB: its median runtime is
within 10% of the fastest configuration and it has the lowest measured peak
memory among those configurations.

The production `ssb inspect --format json` binary then completed the pinned
Next.js repository in 2.27 seconds with 89,423,872 bytes maximum RSS as reported
by `/usr/bin/time -l`. It emitted schema 2 with all 29,073 candidates scanned,
28,403 files indexed, and zero remaining candidates.

## Reproduction

Prepare clean attached clones named `cobra`, `flask`, `django`, and `next.js`
under one directory, each at the manifest commit. Do not install dependencies
or execute repository code.

```bash
SSB_BENCHMARK_ROOT=<fresh-pin-root> \
  go test ./internal/evaluation \
  -run TestPinnedInventoryResourceEnvelope -v

SSB_BENCHMARK_ROOT=<fresh-pin-root> \
  go test ./internal/inventory -run '^$' \
  -bench '^BenchmarkBatchPolicies$' -benchmem -benchtime=1x -count=5
```

Linux amd64 resource-envelope evidence remains required before release. Regular
CI does not clone public repositories and skips these tests when
`SSB_BENCHMARK_ROOT` is unset.
