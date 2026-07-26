# Linux amd64 inventory resource-envelope record

Recorded on 2026-07-26 from the successful [Inventory resource evidence run
30199860874](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/30199860874).

## Immutable run input

- Workflow head:
  `9fea20ffee026d726bac676ad89b15b74b91a457`
- Inventory implementation parent:
  `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Job: [Linux amd64 resource envelope
  89787818769](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/30199860874/job/89787818769)
- Runner: GitHub-hosted `ubuntu-24.04`
- Kernel: Linux 6.17.0-1020-azure, x86_64
- Word size: 64-bit
- Git: 2.54.0
- Go: 1.26.5 (`linux/amd64`)
- Raw GitHub Actions log SHA-256:
  `6f49d1f8897209143d589fc2b520348640bcee6b1c4034fad3fc6e3b80b5648b`
- Inventory contract: `ssb-inventory-v2`, schema 2
- Limits: 40,000 candidate files, 134,217,728 candidate bytes, and
  1,048,576 bytes per file

The workflow asserted `uname -s = Linux`, `uname -m = x86_64`, and a 64-bit
word size before measurement. Each repository was shallow-fetched at its exact
manifest commit with complete blobs and switched to an attached local
`ssb-resource-evidence` branch. No evaluated repository code or dependency was
executed.

## Supported-corpus envelope

`TestPinnedInventoryResourceEnvelope` passed all four exact pins in 2.761
seconds.

| Repository | Candidate files | Candidate bytes | Indexed files | Indexed bytes | Inventory elapsed |
|---|---:|---:|---:|---:|---:|
| Cobra | 66 | 705,271 | 65 | 631,792 | 0.018 s |
| Flask | 235 | 1,814,782 | 230 | 1,474,850 | 0.029 s |
| Django | 7,001 | 45,506,636 | 5,619 | 36,820,618 | 0.673 s |
| Next.js | 29,073 | 111,110,455 | 28,403 | 88,643,646 | 1.462 s |

All four completed within the 10-second per-repository envelope. Next.js
remained below both default candidate limits and returned zero remaining files
and bytes.

## Warm batch-policy sweep

One cold sweep was discarded. Each configuration below then scanned all four
repositories once in each of five warm runs.

| Entries | MiB | Five warm seconds | Median |
|---:|---:|---|---:|
| 32 | 1 | 3.881, 3.850, 3.874, 3.875, 3.865 | 3.874 |
| 128 | 1 | 2.659, 2.663, 2.654, 2.687, 2.661 | 2.661 |
| 512 | 1 | 2.400, 2.456, 2.471, 2.450, 2.420 | 2.450 |
| 32 | 4 | 3.825, 3.815, 3.814, 3.821, 3.800 | 3.815 |
| 128 | 4 | 2.564, 2.550, 2.583, 2.585, 2.560 | 2.564 |
| 512 | 4 | 2.233, 2.249, 2.219, 2.239, 2.220 | 2.233 |
| 32 | 8 | 3.823, 3.798, 3.791, 3.809, 3.780 | 3.798 |
| 128 | 8 | 2.573, 2.546, 2.540, 2.566, 2.547 | 2.547 |
| 512 | 8 | 2.176, 2.198, 2.184, 2.179, 2.199 | 2.184 |
| 32 | 16 | 3.821, 3.827, 3.801, 3.825, 3.821 | 3.821 |
| 128 | 16 | 2.545, 2.544, 2.521, 2.537, 2.509 | 2.537 |
| 512 | 16 | 2.186, 2.172, 2.182, 2.187, 2.233 | 2.186 |

The fastest median was 2.184 seconds at 512 entries and 8 MiB. The 512-entry
4 MiB result was 2.233 seconds, 2.2% slower and therefore within the selected
10% performance band.

## Combined peak RSS

The harness discarded one cold run for each candidate, then sampled the
benchmark process and its direct Git children every 10 milliseconds.

| Entries | MiB | Five combined peak-RSS observations | Median | Maximum |
|---:|---:|---|---:|---:|
| 512 | 4 | 68,923,392; 76,320,768; 77,549,568; 76,394,496; 83,722,240 B | 76,394,496 B | 83,722,240 B |
| 512 | 8 | 75,243,520; 90,791,936; 94,822,400; 78,032,896; 94,445,568 B | 90,791,936 B | 94,822,400 B |
| 512 | 16 | 92,176,384; 91,353,088; 90,849,280; 84,701,184; 85,549,056 B | 90,849,280 B | 92,176,384 B |

All measurements stayed below the 256 MiB envelope. The selected production
policy remains 512 entries and 4 MiB: it is within 10% of the fastest warm
median and has the lowest measured median and maximum combined RSS among the
three sampled candidates.

## Setup retries

Two workflow defects caused setup-only failures before any resource
measurement could start:

- runs 30199742262 and 30199823642 invoked the non-executable harness path
  directly; and
- run 30199832012 used detached benchmark checkouts, which the inventory safety
  gate rejected.

Both failures were corrected in the workflow. They are setup evidence only and
are not included in the measurements above.
