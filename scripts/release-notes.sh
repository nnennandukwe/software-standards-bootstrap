#!/bin/sh

set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: scripts/release-notes.sh vMAJOR.MINOR.PATCH[-PRERELEASE][+BUILD] [CHANGELOG]" >&2
  exit 2
fi

tag=$1
changelog=${2:-CHANGELOG.md}
if ! awk -v tag="$tag" '
function valid_number(value) {
  return value ~ /^(0|[1-9][0-9]*)$/
}
function valid_identifiers(value, reject_numeric_leading_zero, parts, count, part_index) {
  if (value == "") {
    return 0
  }
  count = split(value, parts, ".")
  for (part_index = 1; part_index <= count; part_index++) {
    if (parts[part_index] !~ /^[0-9A-Za-z-]+$/) {
      return 0
    }
    if (reject_numeric_leading_zero && parts[part_index] ~ /^[0-9]+$/ && !valid_number(parts[part_index])) {
      return 0
    }
  }
  return 1
}
BEGIN {
  if (substr(tag, 1, 1) != "v") {
    exit 1
  }
  value = substr(tag, 2)
  build_at = index(value, "+")
  if (build_at) {
    build = substr(value, build_at + 1)
    value = substr(value, 1, build_at - 1)
    if (!valid_identifiers(build, 0)) {
      exit 1
    }
  }
  prerelease_at = index(value, "-")
  if (prerelease_at) {
    prerelease = substr(value, prerelease_at + 1)
    value = substr(value, 1, prerelease_at - 1)
    if (!valid_identifiers(prerelease, 1)) {
      exit 1
    }
  }
  if (split(value, core, ".") != 3 ||
      !valid_number(core[1]) || !valid_number(core[2]) || !valid_number(core[3])) {
    exit 1
  }
  exit 0
}'; then
  echo "invalid release tag: $tag; expected vMAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]" >&2
  exit 2
fi
version=${tag#v}

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
