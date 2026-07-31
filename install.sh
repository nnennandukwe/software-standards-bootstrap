#!/bin/sh

set -eu

REPOSITORY="nnennandukwe/software-standards-bootstrap"
RELEASE_BASE_URL="https://github.com/${REPOSITORY}/releases"

version=${SSB_VERSION:-}
install_dir=${SSB_INSTALL_DIR:-}
skill_dir=${SSB_SKILL_DIR:-}
temp_dir=
target_temp=
skill_temp=
skill_target=
skill_installed=0

usage() {
  cat <<'EOF'
Install Software Standards Bootstrap from a published GitHub release.

Usage:
  install.sh [--version VERSION] [--install-dir DIRECTORY] [--skill-dir DIRECTORY]

Options:
  --version VERSION       Release tag to install, such as v0.1.0.
                          Defaults to the latest published release.
  --install-dir DIRECTORY Destination for the ssb binary.
                          Defaults to $HOME/.local/bin.
  --skill-dir DIRECTORY   Also install the Agent Skill from the same release
                          tag beneath .agents/skills or .claude/skills.
  -h, --help              Show this help.

Environment variables:
  SSB_VERSION             Alternative to --version.
  SSB_INSTALL_DIR         Alternative to --install-dir.
  SSB_SKILL_DIR           Alternative to --skill-dir.
EOF
}

fail() {
  printf 'install.sh: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [ -n "$target_temp" ]; then
    rm -f "$target_temp"
  fi
  if [ -n "$skill_temp" ]; then
    rm -rf "$skill_temp"
  fi
  if [ "$skill_installed" -eq 1 ] && [ -n "$skill_target" ]; then
    rm -rf "$skill_target"
  fi
  if [ -n "$temp_dir" ] && [ -d "$temp_dir" ]; then
    rm -rf "$temp_dir"
  fi
}

trap cleanup 0 HUP INT TERM

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || fail "--version requires a release tag"
      version=$2
      shift 2
      ;;
    --version=*)
      version=${1#*=}
      shift
      ;;
    --install-dir)
      [ "$#" -ge 2 ] || fail "--install-dir requires a directory"
      install_dir=$2
      shift 2
      ;;
    --install-dir=*)
      install_dir=${1#*=}
      shift
      ;;
    --skill-dir)
      [ "$#" -ge 2 ] || fail "--skill-dir requires a directory"
      skill_dir=$2
      shift 2
      ;;
    --skill-dir=*)
      skill_dir=${1#*=}
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      break
      ;;
    *)
      fail "unknown option: $1"
      ;;
  esac
done

[ "$#" -eq 0 ] || fail "unexpected argument: $1"

os_name=$(uname -s)
case "$os_name" in
  Darwin)
    release_os=darwin
    ;;
  Linux)
    release_os=linux
    ;;
  *)
    fail "Unsupported operating system: $os_name. install.sh supports macOS and Linux."
    ;;
esac

arch_name=$(uname -m)
case "$arch_name" in
  x86_64|amd64)
    release_arch=amd64
    ;;
  arm64|aarch64)
    release_arch=arm64
    ;;
  *)
    fail "Unsupported architecture: $arch_name. install.sh supports amd64 and arm64."
    ;;
esac

for command_name in curl tar awk mktemp install; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done

if [ -z "$install_dir" ]; then
  [ -n "${HOME:-}" ] || fail "HOME is not set; pass --install-dir DIRECTORY"
  install_dir=${HOME}/.local/bin
fi

target=${install_dir}/ssb
[ ! -d "$target" ] || fail "install target is a directory: $target"

if [ -n "$skill_dir" ]; then
  skill_target=${skill_dir}/software-standards-bootstrap
  if [ -e "$skill_target" ] || [ -L "$skill_target" ]; then
    fail "Agent Skill already exists at ${skill_target}; review or remove it before installing"
  fi
  for command_name in git cp; do
    command -v "$command_name" >/dev/null 2>&1 ||
      fail "required command not found for --skill-dir: $command_name"
  done
fi

if [ -z "$version" ]; then
  latest_url=$(curl -fsSL -o /dev/null -w '%{url_effective}' "${RELEASE_BASE_URL}/latest") ||
    fail "could not resolve the latest published release"
  latest_url=${latest_url%/}
  version=${latest_url##*/}
fi

case "$version" in
  v[0-9]*.[0-9]*.[0-9]*)
    ;;
  *)
    fail "invalid release tag: $version. Expected a tag such as v0.1.0."
    ;;
esac
case "$version" in
  *[!A-Za-z0-9._+-]*)
    fail "invalid release tag: $version"
    ;;
esac

asset="ssb_${version}_${release_os}_${release_arch}.tar.gz"
download_base="${RELEASE_BASE_URL}/download/${version}"

temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/ssb-install.XXXXXX") ||
  fail "could not create a temporary directory"
archive_path=${temp_dir}/${asset}
checksums_path=${temp_dir}/checksums.txt

curl -fsSL "${download_base}/${asset}" -o "$archive_path" ||
  fail "could not download ${asset} from release ${version}"
curl -fsSL "${download_base}/checksums.txt" -o "$checksums_path" ||
  fail "could not download checksums.txt from release ${version}"

expected_checksum=$(awk -v asset="$asset" '
  $2 == asset { print $1; matches++ }
  END { if (matches != 1) exit 1 }
' "$checksums_path") || fail "checksums.txt does not contain exactly one entry for ${asset}"

if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum=$(sha256sum "$archive_path" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual_checksum=$(shasum -a 256 "$archive_path" | awk '{ print $1 }')
elif command -v openssl >/dev/null 2>&1; then
  actual_checksum=$(openssl dgst -sha256 "$archive_path" | awk '{ print $NF }')
else
  fail "no SHA-256 command found; install sha256sum, shasum, or openssl"
fi

[ "$actual_checksum" = "$expected_checksum" ] ||
  fail "checksum verification failed for ${asset}"

tar -xzf "$archive_path" -C "$temp_dir" ssb ||
  fail "could not extract ssb from ${asset}"
[ -f "${temp_dir}/ssb" ] || fail "${asset} does not contain the ssb binary"
chmod 0755 "${temp_dir}/ssb"
"${temp_dir}/ssb" --help >/dev/null 2>&1 ||
  fail "downloaded ssb binary could not run on ${release_os}/${release_arch}"

if [ -n "$skill_dir" ]; then
  source_checkout=${temp_dir}/source
  git -c advice.detachedHead=false clone --quiet --depth 1 --branch "$version" --single-branch \
    "https://github.com/${REPOSITORY}.git" "$source_checkout" ||
    fail "could not download the Agent Skill from release tag ${version}"
  skill_source=${source_checkout}/skills/software-standards-bootstrap
  [ -f "${skill_source}/SKILL.md" ] ||
    fail "release tag ${version} does not contain the software-standards-bootstrap Agent Skill"
  mkdir -p "$skill_dir" || fail "could not create Agent Skill directory: $skill_dir"
  skill_temp=$(mktemp -d "${skill_dir}/.software-standards-bootstrap.install.XXXXXX") ||
    fail "could not create a temporary Agent Skill directory beneath ${skill_dir}"
  cp -R "${skill_source}/." "$skill_temp" ||
    fail "could not prepare the Agent Skill beneath ${skill_dir}"
fi

mkdir -p "$install_dir" || fail "could not create install directory: $install_dir"
[ ! -d "$target" ] || fail "install target is a directory: $target"
target_temp=$(mktemp "${install_dir}/.ssb-install.XXXXXX") ||
  fail "could not create a temporary file in ${install_dir}"
install -m 0755 "${temp_dir}/ssb" "$target_temp" ||
  fail "could not write the ssb binary to ${install_dir}"

if [ -n "$skill_dir" ]; then
  if [ -e "$skill_target" ] || [ -L "$skill_target" ]; then
    fail "Agent Skill appeared during installation at ${skill_target}; no files were replaced"
  fi
  mv "$skill_temp" "$skill_target" || fail "could not install the Agent Skill to ${skill_target}"
  skill_temp=
  skill_installed=1
fi

mv -f "$target_temp" "$target" || fail "could not replace ${target}"
target_temp=
skill_installed=0

printf 'Installed ssb %s to %s\n' "$version" "$target"
if [ -n "$skill_target" ]; then
  printf 'Installed Agent Skill to %s\n' "$skill_target"
fi
case ":${PATH}:" in
  *:"${install_dir}":*)
    ;;
  *)
    printf '\nAdd ssb to your PATH for this shell:\n  export PATH="%s:$PATH"\n' "$install_dir"
    ;;
esac
