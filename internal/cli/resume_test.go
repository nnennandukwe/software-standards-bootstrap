package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/cli"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"go.yaml.in/yaml/v4"
)

func TestResumeInterruptedGeneration(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	if err := os.Remove(filepath.Join(repo, ".software-standards/manifest.yaml")); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"inspect", "--repo", repo}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "would be overwritten") {
		t.Fatalf("interrupted generation reproduction: exit=%d stderr=%q", code, stderr.String())
	}
	before := git(t, repo, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	stdout.Reset()
	stderr.Reset()
	code := cli.Run([]string{"resume", "--repo", repo, "--format", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resume: exit=%d stderr=%q", code, stderr.String())
	}
	var result struct {
		State             string `json:"state"`
		InventoryReusable bool   `json:"inventory_reusable"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.State != "needs_proposal" || !result.InventoryReusable {
		t.Fatalf("resume result: %s", stdout.String())
	}
	if after := git(t, repo, "status", "--porcelain=v1", "-z", "--untracked-files=all"); after != before {
		t.Fatalf("resume mutated the repository: before=%q after=%q", before, after)
	}
}

func TestResumeCorrectedValidationThroughRendering(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	agents := filepath.Join(repo, "AGENTS.md")
	writeFile(t, agents, "# Developer notes\n\nKeep these manual instructions.\n")
	rule := filepath.Join(repo, ".software-standards/rules/verify-before-merge.md")
	manifest := filepath.Join(repo, ".software-standards/manifest.yaml")
	originalDigest := fileSHA256(t, rule)
	writeFile(t, rule, "# Verify before merge\n")
	replaceResumeText(t, manifest, originalDigest, fileSHA256(t, rule))
	result := resumeResult(t, repo, 1)
	if result.State != "needs_correction" || !result.InventoryReusable || result.ProposalReusable {
		t.Fatalf("invalid source was reused: %#v", result)
	}
	if strings.Contains(strings.Join(result.Repeat, ","), "inspection") || strings.Contains(strings.Join(result.Repeat, ","), "semantic_generation") {
		t.Fatalf("correction repeated finished stages: %#v", result.Repeat)
	}
	runResumeCommand(t, []string{"render", "--repo", repo}, 1)
	if got := readText(t, agents); got != "# Developer notes\n\nKeep these manual instructions.\n" {
		t.Fatalf("validation failure changed manual AGENTS.md: %q", got)
	}
	invalidDigest := fileSHA256(t, rule)
	writeFile(t, rule, "# Verify before merge\n\nRun verification before merging; retain this reviewed edit.\n")
	result = resumeResult(t, repo, 1)
	if result.State != "needs_correction" || !strings.Contains(fmt.Sprint(result.Diagnostics), "SHA-256 does not match") {
		t.Fatalf("stale digest accepted: %#v", result)
	}
	replaceResumeText(t, manifest, invalidDigest, fileSHA256(t, rule))
	result = resumeResult(t, repo, 0)
	if result.State != "ready_to_render" || !result.ProposalReusable || len(result.Files) != 4 {
		t.Fatalf("corrected proposal: %#v", result)
	}
	runResumeCommand(t, []string{"validate", "--repo", repo}, 0)
	before := resumeSnapshot(t, repo)
	runResumeCommand(t, []string{"render", "--repo", repo, "--dry-run"}, 0)
	assertResumeSnapshot(t, repo, before)
	runResumeCommand(t, []string{"render", "--repo", repo}, 0)
	if !strings.HasPrefix(readText(t, agents), "# Developer notes\n\nKeep these manual instructions.\n") {
		t.Fatal("render lost manual notes")
	}
	for range 2 {
		result = resumeResult(t, repo, 0)
		if result.State != "complete" || !result.ProposalReusable || len(result.Repeat) != 1 || result.Repeat[0] != "developer_review" {
			t.Fatalf("completed generation: %#v", result)
		}
	}
	git(t, repo, "add", ".software-standards", "AGENTS.md")
	git(t, repo, "commit", "-m", "reviewed pack fixture")
	result = resumeResult(t, repo, 1)
	if result.State != "existing_pack" || result.ProposalReusable || len(result.Repeat) != 0 {
		t.Fatalf("tracked pack treated as generation: %#v", result)
	}
}

func TestResumeRejectsUntrustedStateWithoutChangingFiles(t *testing.T) {
	cases := []struct {
		name              string
		change            func(*testing.T, string)
		inventoryReusable bool
		state             string
	}{
		{"missing inventory", func(t *testing.T, repo string) { removeResumeFile(t, repo, ".software-standards/inventory.json") }, false, "blocked"},
		{"partial inventory JSON", func(t *testing.T, repo string) {
			writeFile(t, filepath.Join(repo, ".software-standards/inventory.json"), "{\"schema_version\":2,")
		}, false, "blocked"},
		{"incomplete inventory", func(t *testing.T, repo string) {
			replaceResumeText(t, filepath.Join(repo, ".software-standards/inventory.json"), "\"truncated\": false", "\"truncated\": true")
		}, false, "blocked"},
		{"inventory accounting", func(t *testing.T, repo string) {
			replaceResumeText(t, filepath.Join(repo, ".software-standards/inventory.json"), "\"indexed_files\": 2", "\"indexed_files\": 1")
		}, false, "blocked"},
		{"inventory blob identity", func(t *testing.T, repo string) {
			replaceResumeText(t, filepath.Join(repo, ".software-standards/inventory.json"), "\"bytes\": 23", "\"bytes\": 22")
		}, false, "blocked"},
		{"malformed manifest", func(t *testing.T, repo string) {
			writeFile(t, filepath.Join(repo, ".software-standards/manifest.yaml"), "schema: [unfinished\n")
		}, true, "needs_correction"},
		{"unknown manifest field", func(t *testing.T, repo string) {
			path := filepath.Join(repo, ".software-standards/manifest.yaml")
			writeFile(t, path, readText(t, path)+"unknown: true\n")
		}, true, "needs_correction"},
		{"duplicate manifest field", func(t *testing.T, repo string) {
			path := filepath.Join(repo, ".software-standards/manifest.yaml")
			writeFile(t, path, readText(t, path)+"schema: duplicate\n")
		}, true, "needs_correction"},
		{"inconsistent manifest digest", func(t *testing.T, repo string) {
			path := filepath.Join(repo, ".software-standards/report.md")
			writeFile(t, path, readText(t, path)+"\nManual correction retained.\n")
		}, true, "needs_correction"},
		{"missing report", func(t *testing.T, repo string) { removeResumeFile(t, repo, ".software-standards/report.md") }, true, "needs_correction"},
		{"partial proposal", func(t *testing.T, repo string) {
			writeFile(t, filepath.Join(repo, ".software-standards/rules/verify-before-merge.md"), "# unfinished")
		}, true, "needs_correction"},
		{"missing proposal", func(t *testing.T, repo string) {
			removeResumeFile(t, repo, ".software-standards/rules/verify-before-merge.md")
		}, true, "needs_correction"},
		{"unlisted proposal", func(t *testing.T, repo string) {
			writeFile(t, filepath.Join(repo, ".software-standards/rules/manual.md"), "# Manual source\n\nPreserve this unfinished source.\n")
		}, true, "needs_correction"},
		{"changed proposal baseline", func(t *testing.T, repo string) {
			path := filepath.Join(repo, ".software-standards/manifest.yaml")
			baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
			replaceResumeText(t, path, baseline, strings.Repeat("a", 40))
		}, true, "blocked"},
		{"changed pinned HEAD", func(t *testing.T, repo string) {
			writeFile(t, filepath.Join(repo, "new.md"), "new baseline\n")
			git(t, repo, "add", "new.md")
			git(t, repo, "commit", "-m", "advance baseline")
		}, false, "blocked"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			writeFile(t, filepath.Join(repo, "AGENTS.md"), "# Manual instructions\n")
			test.change(t, repo)
			result := resumeResult(t, repo, 1)
			if result.State != test.state || result.InventoryReusable != test.inventoryReusable || result.ProposalReusable || len(result.Diagnostics) == 0 {
				t.Fatalf("untrusted state accepted: %#v", result)
			}
		})
	}
}

func TestResumeInventoryOnlyAndPartialProposal(t *testing.T) {
	for _, proposal := range []bool{false, true} {
		t.Run(fmt.Sprint(proposal), func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			removeResumeFile(t, repo, ".software-standards/manifest.yaml")
			if !proposal {
				removeResumeFile(t, repo, ".software-standards/report.md")
				removeResumeFile(t, repo, ".software-standards/rules/verify-before-merge.md")
			} else {
				writeFile(t, filepath.Join(repo, ".software-standards/rules/verify-before-merge.md"), "# Incomplete manual proposal\n")
			}
			result := resumeResult(t, repo, 0)
			expectedFiles := 1
			if proposal {
				expectedFiles = 3
			}
			if result.State != "needs_proposal" || !result.InventoryReusable || result.ProposalReusable || len(result.Files) != expectedFiles {
				t.Fatalf("unfinished semantic work treated as complete: %#v", result)
			}
		})
	}
}

func TestResumePreservesManualTrackedChangesAndProjectionDrift(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\n// Manual change.\nfunc main() {}\n")
	before := resumeSnapshot(t, repo)
	output, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo}, 2)
	if output != "" || !strings.Contains(diagnostics, "preserve them") {
		t.Fatalf("dirty baseline: stdout=%q stderr=%q", output, diagnostics)
	}
	assertResumeSnapshot(t, repo, before)
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {}\n")
	runResumeCommand(t, []string{"render", "--repo", repo}, 0)
	agents := filepath.Join(repo, "AGENTS.md")
	replaceResumeText(t, agents, "Software Standards Bootstrap", "Manually changed projection")
	result := resumeResult(t, repo, 1)
	if result.State != "blocked" || result.ProposalReusable || strings.Contains(strings.Join(result.Guidance, " "), "Run ssb render") || !strings.Contains(strings.Join(result.Guidance, " "), "Projection checks blocked continuation") || !strings.Contains(fmt.Sprint(result.Diagnostics), "preserve manual edits") {
		t.Fatalf("projection drift: %#v", result)
	}
}

func TestResumeRejectsSymlinkedRetainedPaths(t *testing.T) {
	for _, relative := range []string{".software-standards", ".software-standards/inventory.json", ".software-standards/manifest.yaml", ".software-standards/rules"} {
		t.Run(relative, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			path := filepath.Join(repo, filepath.FromSlash(relative))
			saved := filepath.Join(t.TempDir(), "saved")
			if err := os.Rename(path, saved); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(saved, path); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			result := resumeResult(t, repo, 1)
			if result.ProposalReusable {
				t.Fatalf("symlinked sources reused: %#v", result)
			}
		})
	}
}

func TestResumeHelpAndUsage(t *testing.T) {
	output, diagnostics := runResumeCommand(t, []string{"resume", "--help"}, 0)
	if diagnostics != "" || !strings.Contains(output, "without writing files") {
		t.Fatalf("help: %q %q", output, diagnostics)
	}
	for _, args := range [][]string{{"resume", "--format", "yaml"}, {"resume", "extra"}, {"resume", "--write"}} {
		output, diagnostics = runResumeCommand(t, args, 2)
		if output != "" || !strings.Contains(diagnostics, "ssb resume --help") {
			t.Fatalf("usage: %q %q", output, diagnostics)
		}
	}
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	output, diagnostics = runResumeCommand(t, []string{"resume", "--repo", repo}, 0)
	if diagnostics != "" || !strings.Contains(output, "Inventory reusable: true") || !strings.Contains(output, "Retained: .software-standards/inventory.json sha256:") || !strings.Contains(output, "Repeat: rendering") {
		t.Fatalf("text output: %q %q", output, diagnostics)
	}
}

func resumeResult(t *testing.T, repo string, wantCode int) rulepack.Continuation {
	t.Helper()
	before := resumeSnapshot(t, repo)
	output, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo, "--format", "json"}, wantCode)
	if diagnostics != "" {
		t.Fatalf("JSON diagnostics weren't kept in the response: %q", diagnostics)
	}
	var result rulepack.Continuation
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != 1 {
		t.Fatalf("schema: %d", result.SchemaVersion)
	}
	assertResumeSnapshot(t, repo, before)
	return result
}

func runResumeCommand(t *testing.T, args []string, wantCode int) (string, string) {
	t.Helper()
	var output, diagnostics bytes.Buffer
	if code := cli.Run(args, &output, &diagnostics); code != wantCode {
		t.Fatalf("%v: exit=%d want=%d stdout=%q stderr=%q", args, code, wantCode, output.String(), diagnostics.String())
	}
	return output.String(), diagnostics.String()
}

func replaceResumeText(t *testing.T, path, old, replacement string) {
	t.Helper()
	text := readText(t, path)
	if !strings.Contains(text, old) {
		t.Fatalf("%s missing %q", path, old)
	}
	writeFile(t, path, strings.ReplaceAll(text, old, replacement))
}

func removeResumeFile(t *testing.T, repo, relative string) {
	t.Helper()
	if err := os.Remove(filepath.Join(repo, filepath.FromSlash(relative))); err != nil {
		t.Fatal(err)
	}
}

func resumeSnapshot(t *testing.T, repo string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(repo, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			relative, err := filepath.Rel(repo, path)
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			snapshot[relative] = "directory:" + info.Mode().String()
			return nil
		}
		relative, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			snapshot[relative] = "symlink:" + target
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[relative] = fmt.Sprintf("%v:%s", info.Mode(), content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot["git-status"] = git(t, repo, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	snapshot["git-index-flags"] = git(t, repo, "ls-files", "-v", "-z")
	snapshot["git-index-entries"] = git(t, repo, "ls-files", "--stage", "-z")
	return snapshot
}

func assertResumeSnapshot(t *testing.T, repo string, before map[string]string) {
	t.Helper()
	if after := resumeSnapshot(t, repo); !reflect.DeepEqual(before, after) {
		t.Fatalf("continuation changed retained state:\nbefore=%v\nafter=%v", before, after)
	}
}

func TestResumeInterruptedAssessmentCanBeRetried(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	before := resumeSnapshot(t, repo)
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := &resumeBoundaryContext{Context: parent, action: cancel}
	var output, diagnostics bytes.Buffer
	code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
	if !ctx.triggered.Load() || code != 130 || output.Len() != 0 || diagnostics.String() != "error: interrupted\n" {
		t.Fatalf("interrupted resume: triggered=%t exit=%d stdout=%q stderr=%q", ctx.triggered.Load(), code, output.String(), diagnostics.String())
	}
	assertResumeSnapshot(t, repo, before)
	if result := resumeResult(t, repo, 0); result.State != "ready_to_render" {
		t.Fatalf("retry: %#v", result)
	}
}

func TestResumeDetectsConcurrentChangesAndPreservesThem(t *testing.T) {
	for _, target := range []string{"inventory", "manifest", "report", "rule", "new manifest", "HEAD", "index flags", "staged rename"} {
		t.Run(target, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			paths := map[string]string{"inventory": ".software-standards/inventory.json", "manifest": ".software-standards/manifest.yaml", "report": ".software-standards/report.md", "rule": ".software-standards/rules/verify-before-merge.md"}
			if target == "new manifest" {
				removeResumeFile(t, repo, ".software-standards/manifest.yaml")
			}
			var afterMutation map[string]string
			ctx := &resumeBoundaryContext{Context: t.Context(), action: func() {
				switch target {
				case "index flags":
					git(t, repo, "update-index", "--assume-unchanged", "main.go")
					writeFile(t, filepath.Join(repo, "main.go"), "package main\n\n// Concurrent hidden edit.\nfunc main() {}\n")
				case "staged rename":
					git(t, repo, "config", "diff.renames", "true")
					git(t, repo, "mv", "main.go", "renamed-main.go")
				case "HEAD":
					writeFile(t, filepath.Join(repo, "new.md"), "concurrent change\n")
					git(t, repo, "add", "new.md")
					git(t, repo, "commit", "-m", "concurrent baseline")
				case "new manifest":
					writeFile(t, filepath.Join(repo, ".software-standards/manifest.yaml"), "# Concurrent manual work\n")
				default:
					path := filepath.Join(repo, filepath.FromSlash(paths[target]))
					writeFile(t, path, readText(t, path)+"\nConcurrent manual work.\n")
				}
				afterMutation = resumeSnapshot(t, repo)
			}}
			var output, diagnostics bytes.Buffer
			code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
			expectedReason := "changed during continuation assessment"
			if target == "index flags" {
				expectedReason = "index visibility flags"
			}
			if target == "staged rename" {
				expectedReason = "tracked baseline file"
			}
			if !ctx.triggered.Load() || code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), expectedReason) {
				t.Fatalf("concurrent resume: triggered=%t exit=%d stdout=%q stderr=%q", ctx.triggered.Load(), code, output.String(), diagnostics.String())
			}
			assertResumeSnapshot(t, repo, afterMutation)
		})
	}
}

// Trigger an external actor at the production assessment's final recheck. This
// keeps cancellation and mutation deterministic without process-global hooks.
type resumeBoundaryContext struct {
	context.Context
	action    func()
	boundary  string
	once      sync.Once
	triggered atomic.Bool
}

func (ctx *resumeBoundaryContext) Err() error {
	boundary := ctx.boundary
	if boundary == "" {
		boundary = "/internal/rulepack.AssessContinuation.func"
	}
	programCounters := make([]uintptr, 12)
	count := runtime.Callers(2, programCounters)
	frames := runtime.CallersFrames(programCounters[:count])
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.Function, boundary) {
			ctx.once.Do(func() { ctx.triggered.Store(true); ctx.action() })
			break
		}
		if !more {
			break
		}
	}
	return ctx.Context.Err()
}

func TestResumePreservesEditedTrackedAgentInstructions(t *testing.T) {
	repo, _ := evidenceRepository(t)
	agents := filepath.Join(repo, "AGENTS.md")
	writeFile(t, agents, "# Manual instructions\n\nOriginal instructions.\n")
	git(t, repo, "add", "AGENTS.md")
	git(t, repo, "commit", "-m", "manual baseline instructions")
	baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	writeValidManifestLayoutPack(t, repo, baseline)
	writeFile(t, agents, "# Manual instructions\n\nDeveloper's updated instructions.\n")
	if result := resumeResult(t, repo, 0); result.State != "ready_to_render" {
		t.Fatalf("manual root edits blocked continuation: %#v", result)
	}
	runResumeCommand(t, []string{"render", "--repo", repo}, 0)
	if !strings.HasPrefix(readText(t, agents), "# Manual instructions\n\nDeveloper's updated instructions.\n") {
		t.Fatal("continuation lost tracked manual edits")
	}
}

func TestResumeBaselineCheckDisablesExternalDiff(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	git(t, repo, "config", "diff.external", "ssb-must-not-execute-external-diff")
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\n// Manual edit.\nfunc main() {}\n")
	before := resumeSnapshot(t, repo)
	_, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo}, 2)
	if !strings.Contains(diagnostics, "tracked baseline file") || strings.Contains(diagnostics, "cannot run ssb-must-not-execute") {
		t.Fatalf("baseline guard invoked external diff: %q", diagnostics)
	}
	assertResumeSnapshot(t, repo, before)
}

func TestResumeEmptyProposalStillRequiresDeveloperReview(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	manifest := filepath.Join(repo, ".software-standards/manifest.yaml")
	content := readText(t, manifest)
	start := strings.Index(content, "artifacts:\n")
	if start < 0 {
		t.Fatal("fixture manifest has no artifact index")
	}
	writeFile(t, manifest, content[:start]+"artifacts: []\n")
	removeResumeFile(t, repo, ".software-standards/rules/verify-before-merge.md")
	result := resumeResult(t, repo, 0)
	if result.State != "complete" || !result.ProposalReusable || !reflect.DeepEqual(result.Repeat, []string{"developer_review"}) {
		t.Fatalf("empty proposal: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("empty proposal created a projection: %v", err)
	}
}

func TestResumeProjectionRejectionsStopBeforeRendering(t *testing.T) {
	for _, target := range []string{"malformed markers", "nonregular target"} {
		t.Run(target, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			agents := filepath.Join(repo, "AGENTS.md")
			if target == "malformed markers" {
				writeFile(t, agents, "# Manual text\n\n<!-- software-standards-bootstrap:start -->\n")
			} else if err := os.Mkdir(agents, 0o755); err != nil {
				t.Fatal(err)
			}
			result := resumeResult(t, repo, 1)
			if result.State != "blocked" || result.ProposalReusable || !result.InventoryReusable || len(result.Repeat) == 0 || result.Repeat[0] != "projection_recovery" || strings.Contains(strings.Join(result.Guidance, " "), "Run ssb render") {
				t.Fatalf("projection rejection gave continuation permission: %#v", result)
			}
		})
	}
}

func TestResumeUnreadableSourcesCannotBecomeReusableDuringAssessment(t *testing.T) {
	for _, relative := range []string{".software-standards/report.md", ".software-standards/rules/verify-before-merge.md"} {
		for _, nonregular := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/nonregular=%t", relative, nonregular), func(t *testing.T) {
				repo, baseline := evidenceRepository(t)
				writeValidManifestLayoutPack(t, repo, baseline)
				path := filepath.Join(repo, filepath.FromSlash(relative))
				original, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if nonregular {
					if err := os.Mkdir(path, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				before := resumeSnapshot(t, repo)
				var afterMutation map[string]string
				restore := func() {
					if nonregular {
						if err := os.Remove(path); err != nil {
							t.Fatal(err)
						}
					}
					writeFile(t, path, string(original))
				}
				ctx := &resumeBoundaryContext{Context: t.Context(), boundary: "/internal/rulepack.validateActionablePack", action: func() { restore(); afterMutation = resumeSnapshot(t, repo) }}
				var output, diagnostics bytes.Buffer
				code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
				var result rulepack.Continuation
				if err := json.Unmarshal(output.Bytes(), &result); err != nil {
					t.Fatalf("exit=%d stderr=%q output=%q: %v", code, diagnostics.String(), output.String(), err)
				}
				if code != 1 || result.ProposalReusable || result.State != "needs_correction" {
					t.Fatalf("late source passed assessment: exit=%d result=%#v", code, result)
				}
				if ctx.triggered.Load() {
					assertResumeSnapshot(t, repo, afterMutation)
				} else {
					// Rejecting the initial capture before validation also closes this race.
					assertResumeSnapshot(t, repo, before)
					restore()
				}
				if retry := resumeResult(t, repo, 0); retry.State != "ready_to_render" {
					t.Fatalf("restored source retry: %#v", retry)
				}
			})
		}
	}
}

func TestResumeVisitsTheBaselineInventoryOnce(t *testing.T) {
	for _, command := range []string{"resume", "validate"} {
		t.Run(command, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			ctx := &inventoryVisitContext{Context: t.Context()}
			var output, diagnostics bytes.Buffer
			code := cli.RunContext(ctx, []string{command, "--repo", repo, "--format", "json"}, &output, &diagnostics)
			if code != 0 || ctx.visits.Load() != 2 {
				t.Fatalf("%s visited the two-entry pinned inventory %d times: exit=%d stderr=%q", command, ctx.visits.Load(), code, diagnostics.String())
			}
		})
	}
}

// Count actual inventory candidate visits on this command's context. The fixture
// has two baseline entries; a repeated scan visits those same entries again.
type inventoryVisitContext struct {
	context.Context
	visits atomic.Int32
}

func (ctx *inventoryVisitContext) Err() error {
	programCounter, _, _, ok := runtime.Caller(1)
	if ok && strings.HasSuffix(runtime.FuncForPC(programCounter).Name(), "/internal/inventory.scan") {
		ctx.visits.Add(1)
	}
	return ctx.Context.Err()
}

func TestResumeLateSourceAfterRejectedCaptureStillNeedsFreshAssessment(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	path := filepath.Join(repo, ".software-standards/rules/verify-before-merge.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	var afterMutation map[string]string
	ctx := &resumeBoundaryContext{Context: t.Context(), action: func() {
		writeFile(t, path, string(original))
		afterMutation = resumeSnapshot(t, repo)
	}}
	var output, diagnostics bytes.Buffer
	code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
	var result rulepack.Continuation
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !ctx.triggered.Load() || code != 1 || result.ProposalReusable || len(result.Diagnostics) == 0 {
		t.Fatalf("late source bypassed rejected capture: triggered=%t code=%d result=%#v", ctx.triggered.Load(), code, result)
	}
	assertResumeSnapshot(t, repo, afterMutation)
	if retry := resumeResult(t, repo, 0); retry.State != "ready_to_render" {
		t.Fatalf("fresh assessment: %#v", retry)
	}
}

func TestResumeInventoryProofCannotOutliveRawBytes(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	path := filepath.Join(repo, ".software-standards/inventory.json")
	var afterMutation map[string]string
	ctx := &resumeBoundaryContext{Context: t.Context(), boundary: "/internal/rulepack.validateActionablePack", action: func() {
		writeFile(t, path, readText(t, path)+"\n")
		afterMutation = resumeSnapshot(t, repo)
	}}
	var output, diagnostics bytes.Buffer
	code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
	if !ctx.triggered.Load() || code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "changed during continuation assessment") {
		t.Fatalf("changed raw inventory reused: triggered=%t code=%d stdout=%q stderr=%q", ctx.triggered.Load(), code, output.String(), diagnostics.String())
	}
	assertResumeSnapshot(t, repo, afterMutation)
}

func TestResumeRejectsStagedBaselineRenames(t *testing.T) {
	for _, relative := range []string{"main.go", "AGENTS.md"} {
		t.Run(relative, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			if relative == "AGENTS.md" {
				writeFile(t, filepath.Join(repo, relative), "# Manual baseline instructions\n")
				git(t, repo, "add", relative)
				git(t, repo, "commit", "-m", "manual root baseline")
				baseline = strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
			}
			writeValidManifestLayoutPack(t, repo, baseline)
			git(t, repo, "config", "diff.renames", "true")
			git(t, repo, "mv", relative, "renamed-file")
			before := resumeSnapshot(t, repo)
			output, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo, "--format", "json"}, 2)
			if output != "" || !strings.Contains(diagnostics, "tracked baseline file") {
				t.Fatalf("rename recovery: stdout=%q stderr=%q", output, diagnostics)
			}
			assertResumeSnapshot(t, repo, before)
		})
	}
}

func TestResumeRejectsHiddenBaselineIndexFlags(t *testing.T) {
	for _, flag := range []string{"assume-unchanged", "skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			path := filepath.Join(repo, "main.go")
			original := readText(t, path)
			git(t, repo, "update-index", "--"+flag, "main.go")
			writeFile(t, path, original+"\n// Hidden manual edit.\n")
			before := resumeSnapshot(t, repo)
			output, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo, "--format", "json"}, 2)
			if output != "" || !strings.Contains(diagnostics, "index visibility flags") || !strings.Contains(diagnostics, "preserve edits") {
				t.Fatalf("hidden index recovery: stdout=%q stderr=%q", output, diagnostics)
			}
			assertResumeSnapshot(t, repo, before)
			git(t, repo, "update-index", "--no-"+flag, "main.go")
			runResumeCommand(t, []string{"resume", "--repo", repo}, 2)
			writeFile(t, path, original)
			if retry := resumeResult(t, repo, 0); retry.State != "ready_to_render" {
				t.Fatalf("visible baseline retry: %#v", retry)
			}
		})
	}
}

func TestResumeProjectionTextGivesResumeRecovery(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	runResumeCommand(t, []string{"render", "--repo", repo}, 0)
	rendered := readText(t, filepath.Join(repo, "AGENTS.md"))
	replaceResumeText(t, filepath.Join(repo, "AGENTS.md"), "Software Standards Bootstrap", "Manual projection edit")
	before := resumeSnapshot(t, repo)
	runResumeCommand(t, []string{"validate", "--repo", repo}, 0)
	_, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo}, 1)
	if strings.Contains(diagnostics, "edit the reported source files") || !strings.HasSuffix(diagnostics, "rerun ssb resume --repo PATH.\n") {
		t.Fatalf("projection rejection gave source-only recovery: %q", diagnostics)
	}
	assertResumeSnapshot(t, repo, before)
	writeFile(t, filepath.Join(repo, "AGENTS.md"), rendered)
	if retry := resumeResult(t, repo, 0); retry.State != "complete" {
		t.Fatalf("projection recovery: %#v", retry)
	}
}

func TestResumeRejectsLateChangesToTheProposalSet(t *testing.T) {
	for _, kind := range []string{"unlisted rule", "nested proposal directory", "unbound orientation"} {
		t.Run(kind, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			var afterMutation map[string]string
			ctx := &resumeBoundaryContext{Context: t.Context(), action: func() {
				switch kind {
				case "unlisted rule":
					writeFile(t, filepath.Join(repo, ".software-standards/rules/late-proposal.md"), "# Late manual proposal\n\nPreserve this unfinished candidate.\n")
				case "nested proposal directory":
					if err := os.Mkdir(filepath.Join(repo, ".software-standards/rules/unfinished"), 0o755); err != nil {
						t.Fatal(err)
					}
				case "unbound orientation":
					writeFile(t, filepath.Join(repo, ".software-standards/orientation.yaml"), "schema: ssb.dev/orientation/v1\n")
				}
				afterMutation = resumeSnapshot(t, repo)
			}}
			var output, diagnostics bytes.Buffer
			code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
			if !ctx.triggered.Load() || code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "proposal artifact set changed during continuation assessment") {
				t.Fatalf("late proposal set passed: triggered=%t code=%d stdout=%q stderr=%q", ctx.triggered.Load(), code, output.String(), diagnostics.String())
			}
			assertResumeSnapshot(t, repo, afterMutation)
			runResumeCommand(t, []string{"validate", "--repo", repo}, 1)
			if retry := resumeResult(t, repo, 1); retry.ProposalReusable {
				t.Fatalf("unreconciled candidate reused: %#v", retry)
			}
		})
	}
}

func TestResumeBindsValidationToTheCapturedManifest(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	path := filepath.Join(repo, ".software-standards/manifest.yaml")
	valid := readText(t, path)
	invalid := strings.Replace(valid, "confidence: high", "confidence: low", 1)
	writeFile(t, path, invalid)
	before := resumeSnapshot(t, repo)
	ctx := &validationSwapContext{Context: t.Context(), beforeValidation: func() { writeFile(t, path, valid) }, afterValidation: func() { writeFile(t, path, invalid) }}
	var output, diagnostics bytes.Buffer
	code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
	if !ctx.started.Load() || !ctx.finished.Load() || code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "manifest changed during continuation validation") {
		t.Fatalf("validation used another manifest: started=%t finished=%t code=%d stdout=%q stderr=%q", ctx.started.Load(), ctx.finished.Load(), code, output.String(), diagnostics.String())
	}
	assertResumeSnapshot(t, repo, before)
	runResumeCommand(t, []string{"validate", "--repo", repo}, 1)
}

func TestResumeBindsCapturedSourcesBeforeValidation(t *testing.T) {
	for _, relative := range []string{".software-standards/report.md", ".software-standards/rules/verify-before-merge.md"} {
		t.Run(relative, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			path := filepath.Join(repo, filepath.FromSlash(relative))
			valid := readText(t, path)
			invalid := valid + "\nDeveloper's unfinished edits.\n"
			writeFile(t, path, invalid)
			before := resumeSnapshot(t, repo)
			ctx := &validationSwapContext{Context: t.Context(), beforeValidation: func() { writeFile(t, path, valid) }, afterValidation: func() { writeFile(t, path, invalid) }}
			var output, diagnostics bytes.Buffer
			code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
			var result rulepack.Continuation
			if err := json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatalf("exit=%d stderr=%q output=%q: %v", code, diagnostics.String(), output.String(), err)
			}
			if code != 1 || result.ProposalReusable || result.State != "needs_correction" {
				t.Fatalf("temporary source restoration authorized reuse: started=%t finished=%t exit=%d result=%#v", ctx.started.Load(), ctx.finished.Load(), code, result)
			}
			found := false
			for _, reference := range result.Files {
				if reference.Path == relative && reference.SHA256 == fileSHA256(t, path) {
					found = true
				}
			}
			if !found {
				t.Fatal("correction response lost the retained source identity")
			}
			assertResumeSnapshot(t, repo, before)
			runResumeCommand(t, []string{"validate", "--repo", repo}, 1)
			writeFile(t, path, valid)
			if retry := resumeResult(t, repo, 0); retry.State != "ready_to_render" {
				t.Fatalf("explicit source recovery failed: %#v", retry)
			}
		})
	}
}

func TestResumeBindsManifestToTheCapturedInventory(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	path := filepath.Join(repo, ".software-standards/inventory.json")
	original := readText(t, path)
	alternate := original + "\n"
	manifestPath := filepath.Join(repo, ".software-standards/manifest.yaml")
	validManifest := readText(t, manifestPath)
	writeFile(t, manifestPath, strings.Replace(validManifest, excerptHash(original), excerptHash(alternate), 1))
	before := resumeSnapshot(t, repo)
	ctx := &validationSwapContext{Context: t.Context(), beforeValidation: func() { writeFile(t, path, alternate) }, afterValidation: func() { writeFile(t, path, original) }}
	var output, diagnostics bytes.Buffer
	code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
	var result rulepack.Continuation
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("exit=%d stderr=%q output=%q: %v", code, diagnostics.String(), output.String(), err)
	}
	if code != 1 || result.ProposalReusable || result.State != "needs_correction" {
		t.Fatalf("temporary inventory replacement authorized reuse: started=%t finished=%t exit=%d result=%#v", ctx.started.Load(), ctx.finished.Load(), code, result)
	}
	assertResumeSnapshot(t, repo, before)
	runResumeCommand(t, []string{"validate", "--repo", repo}, 1)
	writeFile(t, manifestPath, validManifest)
	if retry := resumeResult(t, repo, 0); retry.State != "ready_to_render" {
		t.Fatalf("explicit inventory binding recovery failed: %#v", retry)
	}
}

// Simulate an external writer that temporarily corrects validation input and
// restores its earlier bytes before assessment's final identity check.
type validationSwapContext struct {
	context.Context
	beforeValidation func()
	afterValidation  func()
	started          atomic.Bool
	finished         atomic.Bool
}

func (ctx *validationSwapContext) Err() error {
	programCounter, _, _, ok := runtime.Caller(1)
	if ok {
		name := runtime.FuncForPC(programCounter).Name()
		if strings.HasSuffix(name, "/internal/rulepack.validateActionablePack") && ctx.started.CompareAndSwap(false, true) {
			ctx.beforeValidation()
		}
		if strings.Contains(name, "/internal/rulepack.validateActionablePack.func") && ctx.finished.CompareAndSwap(false, true) {
			ctx.afterValidation()
		}
	}
	return ctx.Context.Err()
}

func TestResumePreservesUnsortedRootCoreBytes(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	path := filepath.Join(repo, ".software-standards/manifest.yaml")
	var manifest rulepack.Manifest
	if err := yaml.Load([]byte(readText(t, path)), &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Artifacts[0].Scopes = []string{"**/*"}
	second := manifest.Artifacts[0]
	second.ID = "another-retained-rule"
	second.Path = ".software-standards/rules/another-retained-rule.md"
	secondPath := filepath.Join(repo, filepath.FromSlash(second.Path))
	writeFile(t, secondPath, "# Another retained rule\n\nRun the retained verification before merging.\n")
	second.SHA256 = fileSHA256(t, secondPath)
	manifest.Artifacts = append(manifest.Artifacts, second)
	manifest.RootCore = []string{"verify-before-merge", "another-retained-rule"}
	raw, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(raw))
	result := resumeResult(t, repo, 0)
	if result.State != "ready_to_render" || !result.ProposalReusable {
		t.Fatalf("valid unsorted root core rejected: %#v", result)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("assessment rewrote manifest order: %q %v", got, err)
	}
}

func TestResumeRechecksInputsAfterProjection(t *testing.T) {
	for _, target := range []string{"source", "HEAD", "unlisted native file"} {
		t.Run(target, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			runResumeCommand(t, []string{"render", "--repo", repo}, 0)
			var afterMutation map[string]string
			ctx := &resumeBoundaryContext{Context: t.Context(), boundary: "/internal/render.Apply", action: func() {
				switch target {
				case "source":
					path := filepath.Join(repo, ".software-standards/rules/verify-before-merge.md")
					writeFile(t, path, readText(t, path)+"\nDeveloper's concurrent edit.\n")
				case "HEAD":
					writeFile(t, filepath.Join(repo, "new.md"), "New baseline.\n")
					git(t, repo, "add", "new.md")
					git(t, repo, "commit", "-m", "concurrent baseline")
				default:
					writeFile(t, filepath.Join(repo, ".software-standards/rules/unfinished.md"), "# Unfinished proposal\n")
				}
				afterMutation = resumeSnapshot(t, repo)
			}}
			var output, diagnostics bytes.Buffer
			code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
			if !ctx.triggered.Load() || code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "changed during continuation assessment") {
				t.Fatalf("projection reused stale input: triggered=%t exit=%d stdout=%q stderr=%q", ctx.triggered.Load(), code, output.String(), diagnostics.String())
			}
			assertResumeSnapshot(t, repo, afterMutation)
		})
	}
}

func TestResumeProjectionDiagnosticsUseRepositoryContext(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	writeFile(t, filepath.Join(repo, ".software-standards/routing"), "Developer's file at the routing target.\n")
	before := resumeSnapshot(t, repo)
	result := resumeResult(t, repo, 1)
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Path != "." || result.Diagnostics[0].Field != "projection" || !strings.Contains(result.Diagnostics[0].Message, "routing target") || strings.Contains(result.Diagnostics[0].Message, "AGENTS.md") || !strings.Contains(result.Diagnostics[0].Recovery, ".software-standards/routing") {
		t.Fatalf("routing error attributed to root instructions: %#v", result.Diagnostics)
	}
	assertResumeSnapshot(t, repo, before)
}

func TestResumeIdentifiesUnindexedRepositorySkillsForReview(t *testing.T) {
	for _, tracked := range []bool{false, true} {
		t.Run(fmt.Sprintf("tracked=%t", tracked), func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			path := ".agents/skills/developer-workflow/SKILL.md"
			writeFile(t, filepath.Join(repo, path), "# Developer's workflow\n\nRetain this manual procedure.\n")
			if tracked {
				git(t, repo, "add", path)
				git(t, repo, "commit", "-m", "developer workflow")
				baseline = strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
			}
			writeValidManifestLayoutPack(t, repo, baseline)
			before := resumeSnapshot(t, repo)
			result := resumeResult(t, repo, 0)
			guidance := strings.Join(result.Guidance, " ")
			if !result.ProposalReusable || !strings.Contains(guidance, path) || !strings.Contains(strings.ToLower(guidance), "unindexed repository skill") || !strings.Contains(guidance, "explicit review") {
				t.Fatalf("unindexed skills weren't identified for review: %#v", result)
			}
			for _, reference := range result.Files {
				if reference.Path == path {
					t.Fatal("unindexed skill advertised as validated retained work")
				}
			}
			assertResumeSnapshot(t, repo, before)
			runResumeCommand(t, []string{"validate", "--repo", repo}, 0)
			runResumeCommand(t, []string{"render", "--repo", repo}, 0)
			if complete := resumeResult(t, repo, 0); complete.State != "complete" || !strings.Contains(strings.Join(complete.Guidance, " "), path) {
				t.Fatalf("complete projection lost skill review guidance: %#v", complete)
			}
		})
	}
}

func TestResumeReviewsPartialAndLinkedSkillPathsWithoutFollowingLinks(t *testing.T) {
	for _, target := range []string{"partial", "linked skill", "linked directory", "parent file"} {
		t.Run(target, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			relative := ".agents/skills/unfinished"
			if target == "parent file" {
				relative = ".agents"
				writeFile(t, filepath.Join(repo, relative), "Developer's existing file.\n")
			} else if target == "partial" {
				writeFile(t, filepath.Join(repo, relative, "notes.md"), "Developer's partial work.\n")
			} else {
				if target == "linked directory" {
					relative = ".agents/skills"
				}
				absolute := filepath.Join(repo, filepath.FromSlash(relative))
				if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(t.TempDir(), "missing-target"), absolute); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			before := resumeSnapshot(t, repo)
			result := resumeResult(t, repo, 0)
			if !result.ProposalReusable || !strings.Contains(strings.Join(result.Guidance, " "), relative) {
				t.Fatalf("shared skill path wasn't retained for review: %#v", result)
			}
			assertResumeSnapshot(t, repo, before)
		})
	}
}

func TestResumeRejectsRootRenameWithUntrackedReplacement(t *testing.T) {
	for _, phase := range []string{"initial", "projection"} {
		t.Run(phase, func(t *testing.T) {
			repo, _ := evidenceRepository(t)
			agents := filepath.Join(repo, "AGENTS.md")
			writeFile(t, agents, "# Developer's baseline instructions\n")
			git(t, repo, "add", "AGENTS.md")
			git(t, repo, "commit", "-m", "baseline instructions")
			baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
			writeValidManifestLayoutPack(t, repo, baseline)
			runResumeCommand(t, []string{"render", "--repo", repo}, 0)
			var afterMutation map[string]string
			mutate := func() {
				git(t, repo, "mv", "AGENTS.md", "renamed-instructions.md")
				writeFile(t, agents, "# Developer's new untracked instructions\n")
				afterMutation = resumeSnapshot(t, repo)
			}
			var ctx context.Context = t.Context()
			if phase == "initial" {
				mutate()
			} else {
				ctx = &resumeBoundaryContext{Context: t.Context(), boundary: "/internal/render.Apply", action: mutate}
			}
			var output, diagnostics bytes.Buffer
			code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
			if code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "tracked baseline file") {
				t.Fatalf("untracked replacement hid root rename: exit=%d stdout=%q stderr=%q", code, output.String(), diagnostics.String())
			}
			assertResumeSnapshot(t, repo, afterMutation)
		})
	}
}

func TestResumeRejectsRootTypeChangesAcrossBaselineIndexAndWorktree(t *testing.T) {
	for _, target := range []string{"baseline symlink", "baseline gitlink", "index symlink"} {
		t.Run(target, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			agents := filepath.Join(repo, "AGENTS.md")
			switch target {
			case "baseline symlink":
				if err := os.Symlink("manual-instructions.md", agents); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				git(t, repo, "add", "AGENTS.md")
			case "baseline gitlink":
				git(t, repo, "update-index", "--add", "--cacheinfo", "160000,"+baseline+",AGENTS.md")
				if err := os.Mkdir(agents, 0o755); err != nil {
					t.Fatal(err)
				}
			default:
				writeFile(t, agents, "# Developer's baseline instructions\n")
				git(t, repo, "add", "AGENTS.md")
			}
			git(t, repo, "commit", "-m", "baseline root entry")
			baseline = strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
			writeValidManifestLayoutPack(t, repo, baseline)
			if target == "index symlink" {
				writeFile(t, filepath.Join(repo, "manual-link-target"), "manual-instructions.md\n")
				oid := strings.TrimSpace(git(t, repo, "hash-object", "-w", "manual-link-target"))
				git(t, repo, "update-index", "--cacheinfo", "120000,"+oid+",AGENTS.md")
			} else {
				if target == "baseline symlink" || target == "baseline gitlink" {
					if err := os.Remove(agents); err != nil {
						t.Fatal(err)
					}
				}
				writeFile(t, agents, "# Developer's replacement instructions\n")
				oid := strings.TrimSpace(git(t, repo, "hash-object", "-w", "AGENTS.md"))
				git(t, repo, "update-index", "--add", "--cacheinfo", "100644,"+oid+",AGENTS.md")
			}
			before := resumeSnapshot(t, repo)
			output, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo, "--format", "json"}, 2)
			if output != "" || !strings.Contains(diagnostics, "tracked baseline file") {
				t.Fatalf("root type change bypassed baseline protection: stdout=%q stderr=%q", output, diagnostics)
			}
			assertResumeSnapshot(t, repo, before)
		})
	}
}

func TestResumeRejectsStagedSourcesWithRestoredWorktreeBytes(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	path := filepath.Join(repo, "main.go")
	original := readText(t, path)
	writeFile(t, path, original+"\n// Developer's staged edit.\n")
	git(t, repo, "add", "main.go")
	writeFile(t, path, original)
	before := resumeSnapshot(t, repo)
	output, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo, "--format", "json"}, 2)
	if output != "" || !strings.Contains(diagnostics, "tracked baseline file") {
		t.Fatalf("staged source edit was hidden by restored worktree bytes: stdout=%q stderr=%q", output, diagnostics)
	}
	assertResumeSnapshot(t, repo, before)
}

func TestResumeRejectsRootInterpretationBypasses(t *testing.T) {
	for _, target := range []string{"intent-to-add", "disabled symlinks"} {
		t.Run(target, func(t *testing.T) {
			repo, _ := evidenceRepository(t)
			agents := filepath.Join(repo, "AGENTS.md")
			if target == "disabled symlinks" {
				writeFile(t, agents, "manual-instructions.md")
				oid := strings.TrimSpace(git(t, repo, "hash-object", "-w", "AGENTS.md"))
				git(t, repo, "update-index", "--add", "--cacheinfo", "120000,"+oid+",AGENTS.md")
				git(t, repo, "config", "core.symlinks", "false")
			} else {
				writeFile(t, agents, "# Developer's baseline instructions\n")
				git(t, repo, "add", "AGENTS.md")
			}
			git(t, repo, "commit", "-m", "baseline root entry")
			baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
			writeValidManifestLayoutPack(t, repo, baseline)
			if target == "intent-to-add" {
				git(t, repo, "mv", "AGENTS.md", "renamed-instructions.md")
				writeFile(t, agents, "# Developer's replacement instructions\n")
				git(t, repo, "add", "--intent-to-add", "AGENTS.md")
			}
			before := resumeSnapshot(t, repo)
			output, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo, "--format", "json"}, 2)
			if output != "" || !strings.Contains(diagnostics, "tracked baseline file") {
				t.Fatalf("root interpretation bypassed baseline protection: stdout=%q stderr=%q", output, diagnostics)
			}
			assertResumeSnapshot(t, repo, before)
		})
	}
}

func TestResumePreservesStagedManualRootTextEdits(t *testing.T) {
	repo, _ := evidenceRepository(t)
	agents := filepath.Join(repo, "AGENTS.md")
	writeFile(t, agents, "# Developer's baseline instructions\n")
	git(t, repo, "add", "AGENTS.md")
	git(t, repo, "commit", "-m", "baseline root instructions")
	baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	writeValidManifestLayoutPack(t, repo, baseline)
	writeFile(t, agents, "# Developer's staged instructions\n")
	git(t, repo, "add", "AGENTS.md")
	before := resumeSnapshot(t, repo)
	if result := resumeResult(t, repo, 0); result.State != "ready_to_render" {
		t.Fatalf("manual staged root text was blocked: %#v", result)
	}
	assertResumeSnapshot(t, repo, before)
}

func TestResumeReportsPartialSourceIdentitiesWithoutACompletionClaim(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	removeResumeFile(t, repo, ".software-standards/manifest.yaml")
	writeFile(t, filepath.Join(repo, ".software-standards/rules/verify-before-merge.md"), "# Developer's unfinished proposal\n")
	writeFile(t, filepath.Join(repo, ".software-standards/orientation.yaml"), "partial: [\n")
	before := resumeSnapshot(t, repo)
	result := resumeResult(t, repo, 0)
	if result.State != "needs_proposal" || !result.InventoryReusable || result.ProposalReusable || len(result.Files) != 4 {
		t.Fatalf("partial source identities missing or treated as complete: %#v", result)
	}
	for _, reference := range result.Files {
		if reference.SHA256 != fileSHA256(t, filepath.Join(repo, filepath.FromSlash(reference.Path))) {
			t.Fatalf("partial identity differs from retained bytes: %#v", reference)
		}
	}
	assertResumeSnapshot(t, repo, before)
}

func TestResumeNamesSharedSkillsBeforeProposalCompletion(t *testing.T) {
	for _, missing := range []bool{true, false} {
		t.Run(fmt.Sprintf("missing=%t", missing), func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			code := 1
			if missing {
				removeResumeFile(t, repo, ".software-standards/manifest.yaml")
				code = 0
			} else {
				writeFile(t, filepath.Join(repo, ".software-standards/manifest.yaml"), "unfinished: [\n")
			}
			path := ".agents/skills/developer-workflow/SKILL.md"
			writeFile(t, filepath.Join(repo, path), "# Developer's retained workflow\n")
			before := resumeSnapshot(t, repo)
			result := resumeResult(t, repo, code)
			if !result.InventoryReusable || result.ProposalReusable || !strings.Contains(strings.Join(result.Guidance, " "), path) {
				t.Fatalf("partial proposal omitted shared-skill review guidance: %#v", result)
			}
			for _, reference := range result.Files {
				if reference.Path == path {
					t.Fatal("unindexed shared skill was included in pack identities")
				}
			}
			assertResumeSnapshot(t, repo, before)
		})
	}
}

func TestResumeRejectsProjectionChangesDuringFinalInputRecheck(t *testing.T) {
	for _, relative := range []string{"AGENTS.md", ".software-standards/routing/catalog.md"} {
		t.Run(relative, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			runResumeCommand(t, []string{"render", "--repo", repo}, 0)
			path := filepath.Join(repo, filepath.FromSlash(relative))
			original := readText(t, path)
			var afterMutation map[string]string
			ctx := &resumeBoundaryContext{Context: t.Context(), boundary: "/internal/rulepack.RecheckContinuation", action: func() {
				writeFile(t, path, original+"\nDeveloper's concurrent projection edit.\n")
				afterMutation = resumeSnapshot(t, repo)
			}}
			var output, diagnostics bytes.Buffer
			code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
			if !ctx.triggered.Load() || code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "projection changed during continuation assessment") {
				t.Fatalf("late projection change reported complete: triggered=%t exit=%d stdout=%q stderr=%q", ctx.triggered.Load(), code, output.String(), diagnostics.String())
			}
			assertResumeSnapshot(t, repo, afterMutation)
			writeFile(t, path, original)
			if retry := resumeResult(t, repo, 0); retry.State != "complete" {
				t.Fatalf("reviewed projection restoration failed: %#v", retry)
			}
		})
	}
}

func TestResumeTreatsChangedFileNamesLiterally(t *testing.T) {
	for _, test := range []struct {
		relative string
		staged   bool
	}{
		{relative: ":(literal)secret"},
		{relative: ":(literal)secret", staged: true},
		{relative: "bracket[one].go"},
		{relative: "bracket[one].go", staged: true},
	} {
		t.Run(fmt.Sprintf("%s/staged=%t", test.relative, test.staged), func(t *testing.T) {
			relative := test.relative
			repo, _ := evidenceRepository(t)
			path := filepath.Join(repo, relative)
			if err := os.WriteFile(path, []byte("Developer's baseline file.\n"), 0o644); err != nil {
				t.Skipf("filesystem can't represent fixture filename: %v", err)
			}
			git(t, repo, "--literal-pathspecs", "add", "--", relative)
			git(t, repo, "commit", "-m", "literal filename baseline")
			baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
			writeValidManifestLayoutPack(t, repo, baseline)
			writeFile(t, path, "Developer's edited file.\n")
			if test.staged {
				git(t, repo, "--literal-pathspecs", "add", "--", relative)
				writeFile(t, path, "Developer's baseline file.\n")
			}
			before := resumeSnapshot(t, repo)
			output, diagnostics := runResumeCommand(t, []string{"resume", "--repo", repo, "--format", "json"}, 2)
			if output != "" || !strings.Contains(diagnostics, "tracked baseline file") {
				t.Fatalf("literal filename bypassed source protection: stdout=%q stderr=%q", output, diagnostics)
			}
			assertResumeSnapshot(t, repo, before)
		})
	}
}

func TestResumePartialSourcesPreserveEmptyBytesAndRejectUnsafeReads(t *testing.T) {
	for _, target := range []string{"empty", "symlink", "oversized"} {
		t.Run(target, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeValidManifestLayoutPack(t, repo, baseline)
			removeResumeFile(t, repo, ".software-standards/manifest.yaml")
			relative := ".software-standards/rules/verify-before-merge.md"
			path := filepath.Join(repo, filepath.FromSlash(relative))
			original := readText(t, path)
			code := 1
			switch target {
			case "empty":
				writeFile(t, path, "")
				code = 0
			case "symlink":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(t.TempDir(), "missing-target"), path); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			default:
				writeFile(t, path, strings.Repeat("x", (1<<20)+1))
			}
			before := resumeSnapshot(t, repo)
			result := resumeResult(t, repo, code)
			if !result.InventoryReusable || result.ProposalReusable || code == 1 && len(result.Diagnostics) == 0 {
				t.Fatalf("partial source trust checks failed: %#v", result)
			}
			if target == "empty" {
				found := false
				for _, reference := range result.Files {
					if reference.Path == relative && reference.SHA256 == excerptHash("") {
						found = true
					}
				}
				if !found {
					t.Fatal("empty partial source lost its raw-byte identity")
				}
			}
			assertResumeSnapshot(t, repo, before)
			if target == "symlink" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(t, path, original)
			if retry := resumeResult(t, repo, 0); retry.State != "needs_proposal" || retry.ProposalReusable {
				t.Fatalf("partial source recovery changed completion claim: %#v", retry)
			}
		})
	}
}

func TestResumeRechecksPartialSourceIdentities(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)
	removeResumeFile(t, repo, ".software-standards/manifest.yaml")
	path := filepath.Join(repo, ".software-standards/rules/verify-before-merge.md")
	var afterMutation map[string]string
	ctx := &resumeBoundaryContext{Context: t.Context(), action: func() {
		writeFile(t, path, "# Developer's concurrent partial edit\n")
		afterMutation = resumeSnapshot(t, repo)
	}}
	var output, diagnostics bytes.Buffer
	code := cli.RunContext(ctx, []string{"resume", "--repo", repo, "--format", "json"}, &output, &diagnostics)
	if !ctx.triggered.Load() || code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "retained files changed") {
		t.Fatalf("partial source identity changed unnoticed: triggered=%t exit=%d stdout=%q stderr=%q", ctx.triggered.Load(), code, output.String(), diagnostics.String())
	}
	assertResumeSnapshot(t, repo, afterMutation)
	if retry := resumeResult(t, repo, 0); retry.State != "needs_proposal" || retry.ProposalReusable {
		t.Fatalf("fresh partial-source review failed: %#v", retry)
	}
}
