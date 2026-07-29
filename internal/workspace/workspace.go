// Package workspace resolves and protects the Git repository that ssb inspects.
package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	// ErrPrecondition identifies a repository state the developer must fix
	// before ssb can continue.
	ErrPrecondition = errors.New("repository precondition failed")
	// ErrHistoricalCommit identifies a baseline that is invalid, unresolved,
	// or outside the current repository history.
	ErrHistoricalCommit = errors.New("historical commit is unavailable")
	// ErrGitOperation identifies a Git execution failure rather than a
	// repository-state rejection.
	ErrGitOperation = errors.New("Git operation failed")
)

var gitVersionPattern = regexp.MustCompile(`^git version ([0-9]+)\.([0-9]+)(?:\.|$)`)
var objectIDPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

var runRepositoryGit = runGitInput

// PreconditionError describes an expected, actionable repository-state error.
type PreconditionError struct {
	Problem  string
	Recovery string
}

func (e *PreconditionError) Error() string {
	if e.Recovery == "" {
		return e.Problem
	}
	return e.Problem + "; " + e.Recovery
}

func (e *PreconditionError) Unwrap() error { return ErrPrecondition }

// Repository is a commit-backed Git worktree.
type Repository struct {
	root     string
	baseline string
	gitPath  string
}

// BaselineEntry describes one exact path in the baseline tree.
type BaselineEntry struct {
	Mode string
	Kind string
	OID  string
	Size int64
	Path string
}

// Root returns the canonical worktree root.
func (r *Repository) Root() string { return r.root }

// Baseline returns the full object ID of the commit inspected by ssb.
func (r *Repository) Baseline() string { return r.baseline }

// AtCommit returns a read-only repository view pinned to an explicit commit.
// It is used to validate historical evidence retained by an adopted pack.
func (r *Repository) AtCommit(ctx context.Context, commit string) (*Repository, error) {
	if !objectIDPattern.MatchString(commit) {
		return nil, fmt.Errorf("%w: commit must be a full lowercase object id", ErrHistoricalCommit)
	}
	output, err := r.GitWithInput(
		ctx,
		[]byte(commit+"^{commit}\n"),
		"cat-file",
		"--batch-check=%(objectname) %(objecttype)",
	)
	if err != nil {
		return nil, fmt.Errorf("resolve historical commit %s: %w", commit, err)
	}
	fields := strings.Fields(string(output))
	if len(fields) == 2 && fields[1] == "missing" {
		return nil, fmt.Errorf(
			"%w: historical commit %s cannot be resolved",
			ErrHistoricalCommit,
			commit,
		)
	}
	if len(fields) != 2 || fields[1] != "commit" || !objectIDPattern.MatchString(fields[0]) {
		return nil, fmt.Errorf(
			"%w: Git returned an invalid historical commit response",
			ErrGitOperation,
		)
	}
	resolved := fields[0]
	if _, err := r.Git(ctx, "merge-base", "--is-ancestor", resolved, r.baseline); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("verify historical commit ancestry: %w", err)
		}
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return nil, fmt.Errorf(
				"%w: historical commit %s is not an ancestor of current baseline %s",
				ErrHistoricalCommit,
				resolved,
				r.baseline,
			)
		}
		return nil, fmt.Errorf("verify historical commit ancestry: %w", err)
	}
	return &Repository{root: r.root, baseline: resolved, gitPath: r.gitPath}, nil
}

// OpenForInspect applies the strict, clean-start inspection preconditions.
func OpenForInspect(ctx context.Context, path string) (*Repository, error) {
	repo, err := open(ctx, path, true, "ssb inspect")
	if err != nil {
		return nil, err
	}

	if err := rejectExistingPack(repo.root); err != nil {
		return nil, err
	}
	return repo, nil
}

// OpenForPrune applies clean-snapshot preconditions while requiring an adopted
// standards pack. Unlike OpenForInspect, it never treats the existing pack as
// an overwrite collision.
func OpenForPrune(ctx context.Context, path string) (*Repository, error) {
	repo, err := open(ctx, path, true, "ssb prune inspect")
	if err != nil {
		return nil, err
	}
	if err := requireExistingPack(repo.root); err != nil {
		return nil, err
	}
	if err := repo.RejectUntrackedConfigurations(ctx); err != nil {
		return nil, err
	}
	return repo, nil
}

func rejectExistingPack(root string) error {
	packPath := filepath.Join(root, ".software-standards")
	if _, err := os.Lstat(packPath); err == nil {
		return &PreconditionError{
			Problem:  "an existing .software-standards pack would be overwritten",
			Recovery: "edit or remove the existing pack before starting a new inspection",
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect existing pack: %w", err)
	}
	return nil
}

func requireExistingPack(root string) error {
	packPath := filepath.Join(root, ".software-standards")
	info, err := os.Lstat(packPath)
	if errors.Is(err, os.ErrNotExist) {
		return &PreconditionError{
			Problem:  "prune inspection requires an existing .software-standards pack",
			Recovery: "generate, review, and adopt a standards pack before running ssb prune inspect",
		}
	}
	if err != nil {
		return fmt.Errorf("inspect existing pack: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return &PreconditionError{
			Problem:  ".software-standards must be a real directory for prune inspection",
			Recovery: "move the adopted pack inside the repository and rerun ssb prune inspect",
		}
	}
	return nil
}

// Open resolves a commit-backed repository without requiring a clean worktree.
// Commands that write must validate their own bounded targets after calling it.
func Open(ctx context.Context, path string) (*Repository, error) {
	return open(ctx, path, false, "")
}

func open(ctx context.Context, path string, requireClean bool, recoveryCommand string) (*Repository, error) {
	if path == "" {
		path = "."
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve repository path: %w", err)
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, &PreconditionError{
			Problem:  "Git 2.39 or newer is required",
			Recovery: "install Git and rerun ssb",
		}
	}
	versionCommand := exec.CommandContext(ctx, gitPath, "--version")
	versionCommand.Env = append(os.Environ(), "LC_ALL=C")
	versionOutput, err := versionCommand.Output()
	if err != nil || !supportedGitVersion(strings.TrimSpace(string(versionOutput))) {
		return nil, &PreconditionError{
			Problem:  "Git 2.39 or newer is required",
			Recovery: "upgrade Git and rerun ssb",
		}
	}

	rootOut, err := runGit(ctx, gitPath, absolute, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, &PreconditionError{
			Problem:  fmt.Sprintf("%q is not a Git worktree", path),
			Recovery: "run ssb from a non-bare Git repository or pass --repo PATH",
		}
	}
	root := trimGitLine(rootOut)
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}

	if _, err := runGit(ctx, gitPath, root, "symbolic-ref", "-q", "HEAD"); err != nil {
		return nil, &PreconditionError{
			Problem:  "detached HEAD is not supported",
			Recovery: "switch to a branch before running ssb",
		}
	}
	baselineOut, err := runGit(ctx, gitPath, root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil {
		return nil, &PreconditionError{
			Problem:  "the repository has no commit-backed HEAD",
			Recovery: "create an initial commit before running ssb",
		}
	}

	if requireClean {
		status, statusErr := runGit(ctx, gitPath, root, "status", "--porcelain=v1", "-z", "--untracked-files=no", "--ignore-submodules=untracked")
		if statusErr != nil {
			return nil, fmt.Errorf("read Git status: %w", statusErr)
		}
		if len(status) != 0 {
			return nil, &PreconditionError{
				Problem:  "tracked or staged changes make the inspection baseline ambiguous",
				Recovery: "commit, stash, or restore tracked changes and rerun " + recoveryCommand,
			}
		}
	}

	return &Repository{
		root:     root,
		baseline: trimGitLine(baselineOut),
		gitPath:  gitPath,
	}, nil
}

// VerifyInspectSnapshot rechecks the branch, commit, dirty state, and pack
// collision after inventory reads. A concurrent repository change invalidates
// the result rather than producing mixed-baseline evidence.
func (r *Repository) VerifyInspectSnapshot(ctx context.Context) error {
	if err := r.verifyStableSnapshot(ctx, "inspection"); err != nil {
		return err
	}
	return rejectExistingPack(r.root)
}

// VerifyPruneSnapshot rechecks the immutable prune input while preserving the
// existing adopted pack.
func (r *Repository) VerifyPruneSnapshot(ctx context.Context) error {
	if err := r.verifyStableSnapshot(ctx, "prune inspection"); err != nil {
		return err
	}
	if err := requireExistingPack(r.root); err != nil {
		return err
	}
	return r.RejectUntrackedConfigurations(ctx)
}

// RejectUntrackedConfigurations prevents a current rule or repository skill
// from falling outside the commit-backed review and later being overwritten.
func (r *Repository) RejectUntrackedConfigurations(ctx context.Context) error {
	output, err := r.Git(
		ctx,
		"ls-files", "--others", "-z", "--",
		".software-standards/rules", ".agents/skills",
	)
	if err != nil {
		return fmt.Errorf("list untracked configurations: %w", err)
	}
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		candidate := string(record)
		isRule := strings.HasPrefix(candidate, ".software-standards/rules/") &&
			path.Dir(candidate) == ".software-standards/rules" &&
			strings.HasSuffix(candidate, ".md")
		isSkill := strings.HasPrefix(candidate, ".agents/skills/")
		if isRule || isSkill {
			return &PreconditionError{
				Problem:  "untracked configuration " + strconv.Quote(candidate) + " is outside the commit-backed prune inventory",
				Recovery: "commit, move, or remove the untracked rule or skill file before continuing",
			}
		}
	}
	return nil
}

func (r *Repository) verifyStableSnapshot(ctx context.Context, operation string) error {
	if _, err := r.Git(ctx, "symbolic-ref", "-q", "HEAD"); err != nil {
		return &PreconditionError{
			Problem:  "HEAD became detached during " + operation,
			Recovery: "switch to a branch and rerun ssb inspect",
		}
	}
	current, err := r.Git(ctx, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil || trimGitLine(current) != r.baseline {
		return &PreconditionError{
			Problem:  "HEAD changed during " + operation,
			Recovery: "rerun the command against the new stable baseline",
		}
	}
	status, err := r.Git(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=no", "--ignore-submodules=untracked")
	if err != nil {
		return fmt.Errorf("recheck Git status: %w", err)
	}
	if len(status) != 0 {
		return &PreconditionError{
			Problem:  "tracked or staged files changed during " + operation,
			Recovery: "commit, stash, or restore tracked changes and rerun the command",
		}
	}
	return nil
}

// Git runs Git with the repository fixed by -C and returns stdout. Arguments
// are passed directly to exec.Command; repository paths never enter a shell.
func (r *Repository) Git(ctx context.Context, args ...string) ([]byte, error) {
	return runRepositoryGit(ctx, r.gitPath, r.root, nil, args...)
}

// GitWithInput runs a read-only Git plumbing command with explicit stdin.
func (r *Repository) GitWithInput(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	return runRepositoryGit(ctx, r.gitPath, r.root, input, args...)
}

// ReadBaselineFile returns a tracked regular file from the pinned commit.
// Symlinks, submodules, directories, and path traversal are rejected.
func (r *Repository) ReadBaselineFile(ctx context.Context, filePath string) ([]byte, error) {
	entry, exists, err := r.EntryAtBaseline(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("evidence path %q does not exist at baseline %s", filePath, r.baseline)
	}
	if (entry.Mode != "100644" && entry.Mode != "100755") || entry.Kind != "blob" {
		return nil, fmt.Errorf("evidence path %q is not a tracked regular file", filePath)
	}
	content, err := r.Git(ctx, "cat-file", "blob", entry.OID)
	if err != nil {
		return nil, fmt.Errorf("read evidence path %q: %w", filePath, err)
	}
	return content, nil
}

// EntryAtBaseline resolves one literal repository-relative path.
func (r *Repository) EntryAtBaseline(ctx context.Context, filePath string) (BaselineEntry, bool, error) {
	if filePath == "" || path.IsAbs(filePath) || path.Clean(filePath) != filePath ||
		filePath == "." || filePath == ".." || strings.HasPrefix(filePath, "../") {
		return BaselineEntry{}, false, fmt.Errorf("unsafe baseline path %q", filePath)
	}
	tree, err := r.Git(ctx, "ls-tree", "-z", "-l", r.baseline, "--", filePath)
	if err != nil {
		return BaselineEntry{}, false, err
	}
	records := bytes.Split(tree, []byte{0})
	var record []byte
	for _, candidate := range records {
		if len(candidate) == 0 {
			continue
		}
		if record != nil {
			return BaselineEntry{}, false, fmt.Errorf("baseline path %q is not one exact entry", filePath)
		}
		record = candidate
	}
	if record == nil {
		return BaselineEntry{}, false, nil
	}
	tab := bytes.IndexByte(record, '\t')
	if tab < 0 || string(record[tab+1:]) != filePath {
		return BaselineEntry{}, false, fmt.Errorf("baseline path %q did not resolve exactly", filePath)
	}
	fields := strings.Fields(string(record[:tab]))
	if len(fields) != 4 {
		return BaselineEntry{}, false, fmt.Errorf("parse baseline entry for %q", filePath)
	}
	size := int64(-1)
	if fields[3] != "-" {
		size, err = strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return BaselineEntry{}, false, fmt.Errorf("parse baseline size for %q: %w", filePath, err)
		}
	}
	return BaselineEntry{
		Mode: fields[0],
		Kind: fields[1],
		OID:  fields[2],
		Size: size,
		Path: filePath,
	}, true, nil
}

// HasSubmodulePrefix reports whether any component of relative is a gitlink in
// the pinned baseline.
func (r *Repository) HasSubmodulePrefix(ctx context.Context, relative string) (bool, error) {
	if relative == "" || path.IsAbs(relative) || path.Clean(relative) != relative ||
		relative == "." || relative == ".." || strings.HasPrefix(relative, "../") {
		return false, fmt.Errorf("unsafe repository path %q", relative)
	}
	parts := strings.Split(relative, "/")
	for index := range parts {
		prefix := strings.Join(parts[:index+1], "/")
		entry, exists, err := r.EntryAtBaseline(ctx, prefix)
		if err != nil {
			return false, err
		}
		if exists && entry.Mode == "160000" {
			return true, nil
		}
	}
	return false, nil
}

func runGit(ctx context.Context, gitPath, dir string, args ...string) ([]byte, error) {
	return runGitInput(ctx, gitPath, dir, nil, args...)
}

func runGitInput(ctx context.Context, gitPath, dir string, input []byte, args ...string) ([]byte, error) {
	commandArgs := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, gitPath, commandArgs...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_LITERAL_PATHSPECS=1", "LC_ALL=C")
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf(
				"%w: git %s: %s: %w",
				ErrGitOperation,
				strings.Join(args, " "),
				message,
				ctxErr,
			)
		}
		return nil, fmt.Errorf(
			"%w: git %s: %s: %w",
			ErrGitOperation,
			strings.Join(args, " "),
			message,
			err,
		)
	}
	return stdout.Bytes(), nil
}

func supportedGitVersion(output string) bool {
	match := gitVersionPattern.FindStringSubmatch(output)
	if match == nil {
		return false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return false
	}
	minor, err := strconv.Atoi(match[2])
	if err != nil {
		return false
	}
	return major > 2 || major == 2 && minor >= 39
}

func trimGitLine(output []byte) string {
	if len(output) > 0 && output[len(output)-1] == '\n' {
		output = output[:len(output)-1]
	}
	if len(output) > 0 && output[len(output)-1] == '\r' {
		output = output[:len(output)-1]
	}
	return string(output)
}
