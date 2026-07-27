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

	for _, rule := range rules {
		if strings.Contains(rule.Body, StartMarker) || strings.Contains(rule.Body, EndMarker) {
			return nil, "", "", fmt.Errorf("%w: rule content contains a reserved marker", ErrMarkers)
		}
	}
	orderedRules := append([]rulepack.Rule(nil), rules...)
	sort.SliceStable(orderedRules, func(i, j int) bool {
		leftDirective := directiveRank(effectiveDirective(orderedRules[i]))
		rightDirective := directiveRank(effectiveDirective(orderedRules[j]))
		if leftDirective != rightDirective {
			return leftDirective < rightDirective
		}
		leftImportance := importanceRank(orderedRules[i].Importance)
		rightImportance := importanceRank(orderedRules[j].Importance)
		if leftImportance != rightImportance {
			return leftImportance < rightImportance
		}
		return orderedRules[i].ID < orderedRules[j].ID
	})

	var body strings.Builder
	body.WriteString("## Software Standards Bootstrap\n\n")
	body.WriteString("Generated from `.software-standards/rules/*.md` by `ssb render`. Edit or delete source files, then rerun the command.\n\n")
	fmt.Fprintf(&body, "Baseline: `%s`\n\n", pack.BaselineCommit)
	body.WriteString("### How to apply these standards\n\n")
	body.WriteString("- A rule is active only when its affected path scope matches. For contextual rules, every represented lens dimension must also match; values within one dimension are alternatives.\n")
	body.WriteString("- If the language, framework, task, or affected path is uncertain, load the potentially relevant rule instead of excluding it.\n")
	body.WriteString("- Directives mean: `never` is prohibited, `ask-first` requires developer authorization, `always` is required, and `prefer` is the default when no documented exception or explicit user direction applies.\n")
	body.WriteString("- Legacy v1 rules record no directive; apply their canonical bodies as written without inferring one.\n")
	body.WriteString("- Linked rule files are canonical. This projection is a concise router, not a replacement for their complete guidance.\n")
	body.WriteString("- A mapped command is existing repository evidence only. `ssb` did not execute it, and its presence is not a passing result.\n")

	writeStandingOrders(&body, orderedRules)
	writeMappedCommands(&body, orderedRules)
	writeContextualIndex(&body, orderedRules)

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

func writeStandingOrders(body *strings.Builder, rules []rulepack.Rule) {
	body.WriteString("\n### Standing orders\n")
	wroteAny := false
	legacy := make([]rulepack.Rule, 0)
	for _, rule := range rules {
		if rule.Schema == rulepack.SchemaVersionV1 {
			legacy = append(legacy, rule)
		}
	}
	if len(legacy) != 0 {
		wroteAny = true
		body.WriteString("\n#### Legacy v1 rules (directive not recorded)\n")
		for _, rule := range legacy {
			writeStandingOrder(body, rule)
		}
	}
	for _, directive := range []string{"never", "ask-first", "always", "prefer"} {
		group := make([]rulepack.Rule, 0)
		for _, rule := range rules {
			if isBaseRule(rule) && effectiveDirective(rule) == directive {
				group = append(group, rule)
			}
		}
		if len(group) == 0 {
			continue
		}
		wroteAny = true
		fmt.Fprintf(body, "\n#### %s\n", directiveHeading(directive))
		for _, rule := range group {
			writeStandingOrder(body, rule)
		}
	}
	if !wroteAny {
		body.WriteString("\nNo base standing orders were retained.\n")
	}
}

func writeStandingOrder(body *strings.Builder, rule rulepack.Rule) {
	fmt.Fprintf(body, "\n##### %s (`%s`)\n\n", rule.Title, rule.ID)
	fmt.Fprintf(body, "- Source: [%s](%s)\n", rule.SourcePath, rule.SourcePath)
	fmt.Fprintf(body, "- Scope: %s\n", codeList(rule.Scopes))
	fmt.Fprintf(body, "- Primary topic: `%s`\n", rule.Topic)
	fmt.Fprintf(body, "- Classification: `%s`\n", rule.Classification)
	fmt.Fprintf(body, "- Importance: `%s` (%d/100, `%s`)\n", rule.Importance, rule.Score.Total, rule.Score.Method)
	fmt.Fprintf(body, "- Confidence: `%s`\n", rule.Confidence)
	writeRuleProof(body, rule)
	if len(rule.RelatedSkillIDs) != 0 {
		fmt.Fprintf(body, "- Related skills: %s\n", codeList(rule.RelatedSkillIDs))
	}
	body.WriteString("\n")
	body.WriteString(rule.Body)
	if !strings.HasSuffix(rule.Body, "\n") {
		body.WriteString("\n")
	}
}

func writeMappedCommands(body *strings.Builder, rules []rulepack.Rule) {
	type commandRule struct {
		ID       string
		Coverage string
		Proves   string
		Legacy   bool
	}
	commands := make(map[string][]commandRule)
	for _, rule := range rules {
		proof := mappedProof(rule)
		if proof.Command == "" {
			continue
		}
		commands[proof.Command] = append(commands[proof.Command], commandRule{
			ID:       rule.ID,
			Coverage: proof.Coverage,
			Proves:   proof.Proves,
			Legacy:   proof.Legacy,
		})
	}
	if len(commands) == 0 {
		return
	}
	body.WriteString("\n### Mapped verification commands\n")
	commandNames := make([]string, 0, len(commands))
	for command := range commands {
		commandNames = append(commandNames, command)
	}
	sort.Strings(commandNames)
	for _, command := range commandNames {
		entries := commands[command]
		fmt.Fprintf(body, "\n- `%s` — mapped, not executed by ssb\n", command)
		for _, entry := range entries {
			if entry.Legacy {
				fmt.Fprintf(body, "  - `%s` (rule v1): coverage and bounded proved property were not recorded by the v1 schema.\n", entry.ID)
				continue
			}
			fmt.Fprintf(body, "  - `%s`: `%s` — %s\n", entry.ID, entry.Coverage, entry.Proves)
		}
	}
}

func writeContextualIndex(body *strings.Builder, rules []rulepack.Rule) {
	groups := map[string]map[string][]rulepack.Rule{
		"language":  {},
		"framework": {},
		"task":      {},
	}
	for _, rule := range rules {
		if isBaseRule(rule) {
			continue
		}
		for _, lens := range rule.Lenses {
			if _, supported := groups[lens.Kind]; !supported {
				continue
			}
			groups[lens.Kind][lens.Value] = append(groups[lens.Kind][lens.Value], rule)
		}
	}
	hasContextual := false
	for _, values := range groups {
		if len(values) != 0 {
			hasContextual = true
			break
		}
	}
	if !hasContextual {
		return
	}
	body.WriteString("\n### Contextual rule index\n")
	for _, kind := range []string{"language", "framework", "task"} {
		values := make([]string, 0, len(groups[kind]))
		for value := range groups[kind] {
			values = append(values, value)
		}
		sort.Strings(values)
		for _, value := range values {
			fmt.Fprintf(body, "\n#### %s: `%s`\n", strings.ToUpper(kind[:1])+kind[1:], value)
			for _, rule := range groups[kind][value] {
				fmt.Fprintf(
					body,
					"\n- [%s](%s) (`%s`) — `%s`; lenses: %s; scope: %s; topic: `%s`; importance: `%s`; classification: `%s`\n",
					rule.Title,
					rule.SourcePath,
					rule.ID,
					effectiveDirective(rule),
					lensCodeList(rule.Lenses),
					codeList(rule.Scopes),
					rule.Topic,
					rule.Importance,
					rule.Classification,
				)
				writeRuleProofIndented(body, rule)
				if len(rule.RelatedSkillIDs) != 0 {
					fmt.Fprintf(body, "  - Related skills: %s\n", codeList(rule.RelatedSkillIDs))
				}
			}
		}
	}
}

func writeRuleProof(body *strings.Builder, rule rulepack.Rule) {
	proof := mappedProof(rule)
	if proof.Command != "" && proof.Legacy {
		fmt.Fprintf(body, "- Verification: `%s` (mapped, not executed by ssb; rule v1 did not record coverage or a bounded proved property)\n", proof.Command)
	} else if proof.Command != "" {
		fmt.Fprintf(body, "- Verification: `%s`; `%s`: %s (mapped, not executed by ssb)\n", proof.Command, proof.Coverage, proof.Proves)
	} else if strings.TrimSpace(rule.Verification.ProofGap) != "" {
		fmt.Fprintf(body, "- Proof gap: %s\n", strings.TrimSpace(rule.Verification.ProofGap))
	}
}

func writeRuleProofIndented(body *strings.Builder, rule rulepack.Rule) {
	proof := mappedProof(rule)
	if proof.Command != "" {
		fmt.Fprintf(body, "  - Proof: `%s`: %s; command `%s` is mapped, not executed by ssb.\n", proof.Coverage, proof.Proves, proof.Command)
	} else if strings.TrimSpace(rule.Verification.ProofGap) != "" {
		fmt.Fprintf(body, "  - Proof gap: %s\n", strings.TrimSpace(rule.Verification.ProofGap))
	}
}

type proofMapping struct {
	Command  string
	Coverage string
	Proves   string
	Legacy   bool
}

func mappedProof(rule rulepack.Rule) proofMapping {
	return proofMapping{
		Command:  strings.TrimSpace(rule.Verification.Command),
		Coverage: strings.TrimSpace(rule.Verification.Coverage),
		Proves:   strings.TrimSpace(rule.Verification.Proves),
		Legacy:   rule.Schema != rulepack.SchemaVersionV2,
	}
}

func isBaseRule(rule rulepack.Rule) bool {
	if rule.Schema != rulepack.SchemaVersionV2 {
		return true
	}
	return len(rule.Lenses) == 1 && rule.Lenses[0].Kind == "base"
}

func effectiveDirective(rule rulepack.Rule) string {
	if rule.Schema == rulepack.SchemaVersionV2 {
		return rule.Directive
	}
	return ""
}

func lensCodeList(lenses []rulepack.Lens) string {
	values := make([]string, 0, len(lenses))
	for _, lens := range lenses {
		value := lens.Kind
		if lens.Value != "" {
			value += ":" + lens.Value
		}
		values = append(values, value)
	}
	return codeList(values)
}

func directiveHeading(directive string) string {
	switch directive {
	case "ask-first":
		return "Ask first"
	default:
		return strings.ToUpper(directive[:1]) + directive[1:]
	}
}

func directiveRank(directive string) int {
	switch directive {
	case "never":
		return 0
	case "ask-first":
		return 1
	case "always":
		return 2
	default:
		return 3
	}
}

func importanceRank(importance string) int {
	switch importance {
	case "very-high":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	default:
		return 3
	}
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
