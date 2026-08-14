#!/bin/sh

set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: scripts/release-notes.sh vMAJOR.MINOR.PATCH [CHANGELOG]" >&2
  exit 2
fi

tag=$1
changelog=${2:-CHANGELOG.md}
version=${tag#v}
if [ "$version" = "$tag" ] || [ -z "$version" ]; then
  echo "invalid release tag: $tag; expected a leading v" >&2
  exit 2
fi

awk -v heading="## [$version] - " -v changelog="$changelog" '
  index($0, "## [") == 1 {
    if (capture) {
      exit
    }
    if (index($0, heading) == 1) {
      found = 1
      capture = 1
      next
    }
  }
  capture {
    lines[++count] = $0
    if ($0 ~ /[^[:space:]]/) {
      substantive = 1
    }
  }
  END {
    if (!found) {
      printf "release notes heading not found in %s: %s<date>\n", changelog, heading > "/dev/stderr"
      exit 1
    }
    if (!substantive) {
      printf "release notes section is empty in %s: %s<date>\n", changelog, heading > "/dev/stderr"
      exit 1
    }
    for (line = 1; line <= count; line++) {
      print lines[line]
    }
  }
' "$changelog"
