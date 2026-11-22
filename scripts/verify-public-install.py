"""Verify a public SSB release using a fresh native installation."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import tempfile
from urllib.request import urlopen

REPO = Path(__file__).resolve().parents[1]
SKILL_NAME = "software-standards-bootstrap"
SOURCE_SKILL = f"skills/{SKILL_NAME}"


def git(repo, *args, env=None):
    return subprocess.check_output(["git", "-C", repo, *args], env=env)


def verify_skill(commit, installed):
    # Compare to Git blobs to keep checkout filters out of the expected bytes.
    entries = git(REPO, "ls-tree", "-r", "-z", "--name-only", f"{commit}:{SOURCE_SKILL}")
    expected = {name.decode() for name in entries.split(b"\0") if name}
    if "SKILL.md" not in expected or installed.is_symlink():
        raise ValueError("expected a complete tagged skill and a regular installation")
    actual = set()
    for path in installed.rglob("*"):
        if path.is_symlink():
            raise ValueError(f"unexpected symlink in installed skill: {path}")
        if path.is_dir():
            continue
        name = path.relative_to(installed).as_posix()
        actual.add(name)
        if name not in expected:
            raise ValueError(f"unexpected installed skill file: {name}")
        tagged_bytes = git(REPO, "cat-file", "blob", f"{commit}:{SOURCE_SKILL}/{name}")
        if tagged_bytes != path.read_bytes():
            raise ValueError(f"installed file differs from tagged skill: {name}")
    if actual != expected:
        raise ValueError(f"missing installed skill files: {sorted(expected - actual)}")
    return len(actual)


def verify_recovery(binary, fixture):
    fixture.mkdir()
    git(fixture, "init", "--quiet", "-b", "main")
    git(fixture, "config", "core.autocrlf", "false")
    notes = fixture / "AGENTS.md"
    notes.write_bytes(b"# Developer instructions\n\nKeep manual notes.\n")
    git(fixture, "add", "AGENTS.md")
    fixture_env = dict(os.environ, GIT_AUTHOR_DATE="2000-01-01T00:00:00Z",
                       GIT_COMMITTER_DATE="2000-01-01T00:00:00Z")
    git(
        fixture,
        "-c", "core.hooksPath=disabled-hooks",
        "-c", "commit.gpgsign=false",
        "-c", "user.name=SSB Release",
        "-c", "user.email=release@example.invalid",
        "commit", "--quiet", "-m", "Pin the public installation fixture",
        env=fixture_env,
    )
    baseline = git(fixture, "rev-parse", "HEAD").decode().strip()

    # Capture inspect output before creating a partial pack.
    inventory = subprocess.check_output([binary, "inspect", "--repo", fixture, "--format", "json"])
    pack = fixture / ".software-standards"
    pack.mkdir()
    (pack / "inventory.json").write_bytes(inventory)
    inventory_digest = "sha256:" + hashlib.sha256(inventory).hexdigest()
    expected_files = {".software-standards/inventory.json": inventory_digest}

    def snapshot():
        return {
            p.relative_to(fixture).as_posix(): p.read_bytes()
            for p in fixture.rglob("*")
            if ".git" not in p.relative_to(fixture).parts and p.is_file()
        }

    before = snapshot()
    index = (fixture / ".git/index").read_bytes()
    response = json.loads(subprocess.check_output([binary, "resume", "--repo", fixture, "--format", "json"]))
    retained = {entry["path"]: entry["sha256"] for entry in response["retained_files"]}
    if (response["schema_version"] != 1 or response["state"] != "needs_proposal"
            or response["baseline_commit"] != baseline or response["inventory_reusable"] is not True
            or response["proposal_reusable"] is not False or retained != expected_files):
        raise ValueError(f"unexpected assessment of the interrupted fixture: {response}")
    if (before != snapshot() or (fixture / ".git/index").read_bytes() != index
            or git(fixture, "rev-parse", "HEAD").decode().strip() != baseline):
        raise ValueError("resume changed the fixture, index entries or baseline")
    return {"fixture_commit": baseline, "resume_state": response["state"], "inventory": inventory_digest}


def verify_public_install(tag, root):
    commit = git(REPO, "rev-parse", "--verify", f"refs/tags/{tag}^{{commit}}").decode().strip()
    windows = os.name == "nt"
    name = "install.ps1" if windows else "install.sh"
    url = f"https://raw.githubusercontent.com/nnennandukwe/{SKILL_NAME}/{commit}/{name}"
    with urlopen(url, timeout=60) as response:
        installer = response.read()
    if installer != git(REPO, "cat-file", "blob", f"{commit}:{name}"):
        raise ValueError(f"public {name} differs from the tagged Git blob")
    script = root / name
    script.write_bytes(installer)
    bin_dir, skill_dir = root / "bin", root / "skills"
    if windows:
        command = ["powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
                   "-File", script, "-Version", tag, "-InstallDir", bin_dir, "-SkillDir", skill_dir]
    else:
        command = ["sh", script, "--version", tag, "--install-dir", bin_dir, "--skill-dir", skill_dir]
    subprocess.run(command, check=True)
    binary = bin_dir / ("ssb.exe" if windows else "ssb")
    identity = f"ssb {tag} (commit {commit})"
    for argument in ("version", "--version"):
        if subprocess.check_output([binary, argument]).decode().strip() != identity:
            raise ValueError(f"installed binary returned unexpected identity for {argument}")
    help_text = subprocess.check_output([binary, "--help"])
    if b"resume" not in help_text:
        raise ValueError("installed binary help omits resume")
    count = verify_skill(commit, skill_dir / SKILL_NAME)
    result = verify_recovery(binary, root / "fixture")
    result.update({
        "identity": identity,
        "platform": platform.platform(),
        "machine": platform.machine(),
        "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
        "verified_skill_files": count,
        "skill_tree": git(REPO, "rev-parse", f"{commit}:{SOURCE_SKILL}").decode().strip(),
    })
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag", help="published release tag, such as v0.3.0")
    with tempfile.TemporaryDirectory(prefix="ssb-public-install-") as dest:
        verify_public_install(parser.parse_args().tag, Path(dest))
