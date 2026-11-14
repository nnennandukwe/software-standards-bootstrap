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
			if result.State != "needs_proposal" || !result.InventoryReusable || result.ProposalReusable || len(result.Files) != 1 {
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
