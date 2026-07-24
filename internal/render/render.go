// Package render projects validated rule sources into one bounded AGENTS.md
// managed section while preserving all surrounding bytes.
package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const (
	StartMarker = "<!-- software-standards-bootstrap:start -->"
	EndMarker   = "<!-- software-standards-bootstrap:end -->"
)

var (
	ErrDrift        = errors.New("managed AGENTS.md section drifted")
	ErrMarkers      = errors.New("managed AGENTS.md markers are malformed")
	ErrUnsafeTarget = errors.New("unsafe AGENTS.md target")
)

// Result describes the proposed or written projection.
type Result struct {
	Path          string `json:"path"`
	Changed       bool   `json:"changed"`
	DryRun        bool   `json:"dry_run"`
	SourceDigest  string `json:"source_digest"`
	ContentDigest string `json:"content_digest"`
	Content       []byte `json:"-"`
}

// Apply renders a valid pack. It never repairs malformed markers or overwrites
// a symlink. Dry runs return the complete proposed file without writing it.
func Apply(repo *workspace.Repository, pack rulepack.Pack, dryRun bool) (Result, error) {
	target := filepath.Join(repo.Root(), "AGENTS.md")
	existing, mode, exists, err := readTarget(target)
	if err != nil {
		return Result{}, err
	}

	section, sourceDigest, contentDigest, err := buildSection(pack)
	if err != nil {
		return Result{}, err
	}
	next, err := replaceManagedSection(existing, section)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		Path:          "AGENTS.md",
		Changed:       !bytes.Equal(existing, next),
		DryRun:        dryRun,
		SourceDigest:  sourceDigest,
		ContentDigest: contentDigest,
		Content:       next,
	}
	if dryRun || !result.Changed {
		return result, nil
	}
	if err := writeAtomic(target, existing, next, mode, exists); err != nil {
		return Result{}, err
	}
	return result, nil
}

func buildSection(pack rulepack.Pack) ([]byte, string, string, error) {
	rules := append([]rulepack.Rule(nil), pack.Rules...)
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].ID == rules[j].ID {
			return rules[i].SourcePath < rules[j].SourcePath
		}
		return rules[i].ID < rules[j].ID
	})
	sourceState := struct {
		Baseline string          `json:"baseline"`
		Rules    []rulepack.Rule `json:"rules"`
	}{
		Baseline: pack.BaselineCommit,
		Rules:    rules,
	}
	canonical, err := json.Marshal(sourceState)
	if err != nil {
		return nil, "", "", fmt.Errorf("encode rule source digest: %w", err)
	}
	sourceDigest := digest(canonical)

	var body strings.Builder
	body.WriteString("## Software Standards Bootstrap\n\n")
	body.WriteString("Generated from `.software-standards/rules/*.md` by `ssb render`. Edit or delete source files, then rerun the command.\n\n")
	fmt.Fprintf(&body, "Baseline: `%s`\n", pack.BaselineCommit)
	for _, rule := range rules {
		fmt.Fprintf(&body, "\n### %s (`%s`)\n\n", rule.Title, rule.ID)
		fmt.Fprintf(&body, "- Source: `%s`\n", rule.SourcePath)
		fmt.Fprintf(&body, "- Scope: %s\n", codeList(rule.Scopes))
		fmt.Fprintf(&body, "- Primary topic: `%s`\n", rule.Topic)
		fmt.Fprintf(&body, "- Classification: `%s`\n", rule.Classification)
		fmt.Fprintf(&body, "- Importance: `%s` (%d/100, `%s`)\n", rule.Importance, rule.Score.Total, rule.Score.Method)
		fmt.Fprintf(&body, "- Confidence: `%s`\n", rule.Confidence)
		if strings.TrimSpace(rule.Verification.Command) != "" {
			fmt.Fprintf(&body, "- Existing verification: `%s` (cited, not executed by ssb)\n", rule.Verification.Command)
		} else if strings.TrimSpace(rule.Verification.ProofGap) != "" {
			fmt.Fprintf(&body, "- Proof gap: %s\n", strings.TrimSpace(rule.Verification.ProofGap))
		}
		if len(rule.RelatedSkillIDs) != 0 {
			fmt.Fprintf(&body, "- Related skills: %s\n", codeList(rule.RelatedSkillIDs))
		}
		body.WriteString("\n")
		body.WriteString(rule.Body)
		if !strings.HasSuffix(rule.Body, "\n") {
			body.WriteString("\n")
		}
	}
	bodyBytes := []byte(body.String())
	if bytes.Contains(bodyBytes, []byte(StartMarker)) || bytes.Contains(bodyBytes, []byte(EndMarker)) {
		return nil, "", "", fmt.Errorf("%w: rule content contains a reserved marker", ErrMarkers)
	}

	sourceLine := "<!-- source-digest: " + sourceDigest + " -->"
	contentDigest := digest(append([]byte(sourceLine+"\n"), bodyBytes...))
	var section strings.Builder
	section.WriteString(StartMarker)
	section.WriteString("\n")
	section.WriteString(sourceLine)
	section.WriteString("\n")
	fmt.Fprintf(&section, "<!-- content-digest: %s -->\n", contentDigest)
	section.Write(bodyBytes)
	section.WriteString(EndMarker)
	return []byte(section.String()), sourceDigest, contentDigest, nil
}

func replaceManagedSection(existing, section []byte) ([]byte, error) {
	startCount := bytes.Count(existing, []byte(StartMarker))
	endCount := bytes.Count(existing, []byte(EndMarker))
	if startCount == 0 && endCount == 0 {
		next := append([]byte(nil), existing...)
		switch {
		case len(next) == 0:
		case bytes.HasSuffix(next, []byte("\n\n")):
		case bytes.HasSuffix(next, []byte("\n")):
			next = append(next, '\n')
		default:
			next = append(next, '\n', '\n')
		}
		next = append(next, section...)
		next = append(next, '\n')
		return next, nil
	}
	if startCount != 1 || endCount != 1 {
		return nil, fmt.Errorf("%w: expected exactly one start and one end marker", ErrMarkers)
	}
	start := bytes.Index(existing, []byte(StartMarker))
	end := bytes.Index(existing, []byte(EndMarker))
	if end < start+len(StartMarker) {
		return nil, fmt.Errorf("%w: end marker appears before start marker", ErrMarkers)
	}
	if err := verifyExistingSection(existing[start : end+len(EndMarker)]); err != nil {
		return nil, err
	}
	next := make([]byte, 0, len(existing)-((end+len(EndMarker))-start)+len(section))
	next = append(next, existing[:start]...)
	next = append(next, section...)
	next = append(next, existing[end+len(EndMarker):]...)
	return next, nil
}

func verifyExistingSection(section []byte) error {
	prefix := []byte(StartMarker + "\n<!-- source-digest: ")
	if !bytes.HasPrefix(section, prefix) || !bytes.HasSuffix(section, []byte(EndMarker)) {
		return fmt.Errorf("%w: managed section metadata is missing", ErrMarkers)
	}
	rest := section[len(prefix):]
	sourceEnd := bytes.Index(rest, []byte(" -->\n"))
	if sourceEnd < 0 {
		return fmt.Errorf("%w: source digest metadata is malformed", ErrMarkers)
	}
	sourceDigest := string(rest[:sourceEnd])
	if !validDigest(sourceDigest) {
		return fmt.Errorf("%w: source digest metadata is malformed", ErrMarkers)
	}
	rest = rest[sourceEnd+len(" -->\n"):]
	contentPrefix := []byte("<!-- content-digest: ")
	if !bytes.HasPrefix(rest, contentPrefix) {
		return fmt.Errorf("%w: content digest metadata is missing", ErrMarkers)
	}
	rest = rest[len(contentPrefix):]
	contentEnd := bytes.Index(rest, []byte(" -->\n"))
	if contentEnd < 0 {
		return fmt.Errorf("%w: content digest metadata is malformed", ErrMarkers)
	}
	recorded := string(rest[:contentEnd])
	if !validDigest(recorded) {
		return fmt.Errorf("%w: content digest metadata is malformed", ErrMarkers)
	}
	bodyWithEnd := rest[contentEnd+len(" -->\n"):]
	body := bodyWithEnd[:len(bodyWithEnd)-len(EndMarker)]
	actual := digest(append([]byte("<!-- source-digest: "+sourceDigest+" -->\n"), body...))
	if recorded != actual {
		return fmt.Errorf("%w: edit or delete rule source files instead of editing the generated section", ErrDrift)
	}
	return nil
}

func readTarget(target string) ([]byte, os.FileMode, bool, error) {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0o644, false, nil
	}
	if err != nil {
		return nil, 0, false, fmt.Errorf("inspect AGENTS.md: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, 0, false, fmt.Errorf("%w: AGENTS.md must be a real regular file", ErrUnsafeTarget)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, 0, false, fmt.Errorf("read AGENTS.md: %w", err)
	}
	return content, info.Mode().Perm(), true, nil
}

func writeAtomic(target string, expected, content []byte, mode os.FileMode, existed bool) (returnErr error) {
	parent := filepath.Dir(target)
	temp, err := os.CreateTemp(parent, ".ssb-agents-*")
	if err != nil {
		return fmt.Errorf("create staged AGENTS.md: %w", err)
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()
	if err := temp.Chmod(mode); err != nil {
		return fmt.Errorf("set staged AGENTS.md permissions: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		return fmt.Errorf("write staged AGENTS.md: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync staged AGENTS.md: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close staged AGENTS.md: %w", err)
	}

	current, statErr := os.Lstat(target)
	if existed {
		if statErr != nil || current.Mode()&os.ModeSymlink != 0 || !current.Mode().IsRegular() {
			return fmt.Errorf("%w: AGENTS.md changed while rendering", ErrUnsafeTarget)
		}
		currentContent, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(currentContent, expected) {
			return fmt.Errorf("%w: AGENTS.md changed while rendering", ErrDrift)
		}
	} else if statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("%w: AGENTS.md appeared while rendering", ErrUnsafeTarget)
	}
	if err := os.Rename(tempPath, target); err != nil {
		return fmt.Errorf("replace AGENTS.md atomically: %w", err)
	}
	return nil
}

func codeList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && value == strings.ToLower(value)
}
