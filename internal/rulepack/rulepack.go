// Package rulepack validates the editable, evidence-backed standards pack.
package rulepack

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const (
	SchemaVersionV1 = "ssb.dev/rule/v1"
	SchemaVersionV2 = "ssb.dev/rule/v2"
	ScoreMethod     = "ssb-score-v1"
	topicRecovery   = "use one primary topic: architecture, compatibility, compliance, correctness, developer-experience, documentation, maintainability, operability, performance, quality, reliability, security, or testability"
)

var (
	stableIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	digestPattern   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	supportedTopics = map[string]struct{}{
		"architecture":         {},
		"compatibility":        {},
		"compliance":           {},
		"correctness":          {},
		"developer-experience": {},
		"documentation":        {},
		"maintainability":      {},
		"operability":          {},
		"performance":          {},
		"quality":              {},
		"reliability":          {},
		"security":             {},
		"testability":          {},
	}
	supportedTasks = map[string]struct{}{
		"implementation": {},
		"review":         {},
		"testing":        {},
		"security":       {},
		"documentation":  {},
		"release":        {},
	}
	supportedDirectives = map[string]struct{}{
		"always":    {},
		"ask-first": {},
		"never":     {},
		"prefer":    {},
	}
)

// Diagnostic is a recoverable, file-specific validation problem.
type Diagnostic struct {
	Path     string `json:"path"`
	Line     int    `json:"line,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Recovery string `json:"recovery,omitempty"`
}

// Score records transparent ranking arithmetic.
type Score struct {
	Method  string       `yaml:"method" json:"method"`
	Total   int          `yaml:"total" json:"total"`
	Factors ScoreFactors `yaml:"factors" json:"factors"`
}

// ScoreFactors are the bounded ssb-score-v1 inputs.
type ScoreFactors struct {
	Prevalence    int `yaml:"prevalence" json:"prevalence"`
	Consistency   int `yaml:"consistency" json:"consistency"`
	Authority     int `yaml:"authority" json:"authority"`
	Risk          int `yaml:"risk" json:"risk"`
	Applicability int `yaml:"applicability" json:"applicability"`
}

// Evidence points to an exact line range in the baseline commit.
type Evidence struct {
	Path          string `yaml:"path" json:"path"`
	Lines         string `yaml:"lines" json:"lines"`
	ExcerptSHA256 string `yaml:"excerpt_sha256" json:"excerpt_sha256"`
	Authoritative bool   `yaml:"authoritative,omitempty" json:"authoritative,omitempty"`
}

// Lens identifies one context dimension used to select a rule. Values within
// one kind are alternatives; represented kinds are matched together.
type Lens struct {
	Kind  string `yaml:"kind" json:"kind"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

// Verification either cites an existing repository check or records a proof
// gap. ssb never executes the command.
type Verification struct {
	Command  string    `yaml:"command,omitempty" json:"command,omitempty"`
	Source   *Evidence `yaml:"source,omitempty" json:"source,omitempty"`
	ProofGap string    `yaml:"proof_gap,omitempty" json:"proof_gap,omitempty"`
	Coverage string    `yaml:"coverage,omitempty" json:"coverage,omitempty"`
	Proves   string    `yaml:"proves,omitempty" json:"proves,omitempty"`
}

// Rule is one editable rule source file plus its exact Markdown body.
type Rule struct {
	Schema          string       `yaml:"schema" json:"schema"`
	ID              string       `yaml:"id" json:"id"`
	Title           string       `yaml:"title" json:"title"`
	Topic           string       `yaml:"topic" json:"topic"`
	Lenses          []Lens       `yaml:"lenses,omitempty" json:"lenses,omitempty"`
	Directive       string       `yaml:"directive,omitempty" json:"directive,omitempty"`
	Scopes          []string     `yaml:"scopes" json:"scopes"`
	Classification  string       `yaml:"classification" json:"classification"`
	Importance      string       `yaml:"importance" json:"importance"`
	Score           Score        `yaml:"score" json:"score"`
	Confidence      string       `yaml:"confidence" json:"confidence"`
	BaselineCommit  string       `yaml:"baseline_commit" json:"baseline_commit"`
	Evidence        []Evidence   `yaml:"evidence" json:"evidence"`
	Verification    Verification `yaml:"verification" json:"verification"`
	RelatedSkillIDs []string     `yaml:"related_skills,omitempty" json:"related_skills,omitempty"`

	SourcePath string `yaml:"-" json:"source_path"`
	Body       string `yaml:"-" json:"body"`
}

// Skill is a validated portable Agent Skill referenced by a rule.
type Skill struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Topic       string `json:"topic"`
	SourcePath  string `json:"source_path"`
	Body        string `json:"body"`
}

// Pack contains parsed artifacts even when diagnostics are returned. Consumers
// must not render or create an ADR unless diagnostics is empty.
type Pack struct {
	BaselineCommit string  `json:"baseline_commit"`
	AssessmentPath string  `json:"assessment_path"`
	Assessment     string  `json:"assessment"`
	Rules          []Rule  `json:"rules"`
	Skills         []Skill `json:"skills"`
}

type skillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
}

// Validate parses the current editable pack and verifies all evidence against
// the repository's pinned HEAD commit. Validation never writes files.
func Validate(ctx context.Context, repo *workspace.Repository) (Pack, []Diagnostic, error) {
	pack := Pack{
		BaselineCommit: repo.Baseline(),
		AssessmentPath: ".software-standards/assessment.md",
		Rules:          make([]Rule, 0),
		Skills:         make([]Skill, 0),
	}
	diagnostics := make([]Diagnostic, 0)

	packRootPath := filepath.Join(repo.Root(), ".software-standards")
	if info, err := os.Lstat(packRootPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			diagnostics = append(diagnostics, diagnostic(".software-standards", "file", ".software-standards must be a real directory, not a symlink", "move the editable pack inside the repository"))
			return pack, diagnostics, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Pack{}, nil, fmt.Errorf("inspect .software-standards: %w", err)
	}

	assessmentPath := filepath.Join(repo.Root(), filepath.FromSlash(pack.AssessmentPath))
	assessment, assessmentDiagnostics, err := readRequiredRegularFile(assessmentPath, pack.AssessmentPath)
	if err != nil {
		return Pack{}, nil, err
	}
	diagnostics = append(diagnostics, assessmentDiagnostics...)
	if len(assessment) != 0 {
		pack.Assessment = string(assessment)
		if strings.TrimSpace(pack.Assessment) == "" {
			diagnostics = append(diagnostics, diagnostic(pack.AssessmentPath, "assessment", "assessment must not be empty", "record repository context and candidate rationale"))
		}
	}

	rulesDirPath := filepath.Join(repo.Root(), ".software-standards", "rules")
	info, err := os.Lstat(rulesDirPath)
	if errors.Is(err, os.ErrNotExist) {
		diagnostics = append(diagnostics, diagnostic(".software-standards/rules", "rules", "rule directory does not exist", "create one Markdown file per retained rule"))
		return pack, diagnostics, nil
	}
	if err != nil {
		return Pack{}, nil, fmt.Errorf("inspect rule directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		diagnostics = append(diagnostics, diagnostic(".software-standards/rules", "rules", "rule directory must be a real directory, not a symlink", "replace it with a directory inside the repository"))
		return pack, diagnostics, nil
	}
	entries, err := os.ReadDir(rulesDirPath)
	if err != nil {
		return Pack{}, nil, fmt.Errorf("read rule directory: %w", err)
	}
	ruleFiles := make([]string, 0)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			ruleFiles = append(ruleFiles, entry.Name())
		}
	}
	sort.Strings(ruleFiles)
	if len(ruleFiles) == 0 {
		diagnostics = append(diagnostics, diagnostic(".software-standards/rules", "rules", "no rule files found", "retain at least one scored candidate or keep candidates in the assessment only"))
		return pack, diagnostics, nil
	}

	skillCache := make(map[string]Skill)
	seenRuleIDs := make(map[string]string)
	for _, fileName := range ruleFiles {
		relative := path.Join(".software-standards/rules", fileName)
		absolute := filepath.Join(rulesDirPath, fileName)
		data, fileDiagnostics, readErr := readRequiredRegularFile(absolute, relative)
		if readErr != nil {
			return Pack{}, nil, readErr
		}
		diagnostics = append(diagnostics, fileDiagnostics...)
		if len(data) == 0 {
			continue
		}
		rule, parseDiagnostic := parseRule(relative, data)
		if parseDiagnostic != nil {
			diagnostics = append(diagnostics, *parseDiagnostic)
			continue
		}
		pack.Rules = append(pack.Rules, rule)
		diagnostics = append(diagnostics, validateRule(ctx, repo, rule, fileName)...)

		if prior, exists := seenRuleIDs[rule.ID]; exists {
			diagnostics = append(diagnostics, diagnostic(relative, "id", fmt.Sprintf("duplicate rule id %q also used by %s", rule.ID, prior), "give every rule a stable unique id"))
		} else {
			seenRuleIDs[rule.ID] = relative
		}
		for _, skillID := range rule.RelatedSkillIDs {
			if _, loaded := skillCache[skillID]; loaded {
				continue
			}
			skill, skillDiagnostics, loadErr := loadSkill(repo.Root(), skillID)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, skillDiagnostics...)
			if skill.ID != "" {
				skillCache[skillID] = skill
			}
		}
	}
	for _, skill := range skillCache {
		pack.Skills = append(pack.Skills, skill)
	}
	sort.Slice(pack.Skills, func(i, j int) bool { return pack.Skills[i].ID < pack.Skills[j].ID })
	return pack, diagnostics, nil
}

func parseRule(relative string, data []byte) (Rule, *Diagnostic) {
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		item := diagnostic(relative, "frontmatter", err.Error(), "add strict YAML frontmatter between --- markers")
		return Rule{}, &item
	}
	var rule Rule
	if err := yaml.Load(frontmatter, &rule, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		line := 1
		var loadError *yaml.LoadError
		if errors.As(err, &loadError) && loadError.Mark.Line > 0 {
			line = loadError.Mark.Line + 1
		}
		item := Diagnostic{
			Path:     relative,
			Line:     line,
			Field:    "frontmatter",
			Message:  err.Error(),
			Recovery: "remove unknown or duplicate fields and use the ssb.dev/rule/v1 or ssb.dev/rule/v2 schema",
		}
		return Rule{}, &item
	}
	rule.SourcePath = relative
	rule.Body = string(body)
	return rule, nil
}

func validateRule(ctx context.Context, repo *workspace.Repository, rule Rule, fileName string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(rule.SourcePath, field, message, recovery))
	}

	if rule.Schema != SchemaVersionV1 && rule.Schema != SchemaVersionV2 {
		add("schema", fmt.Sprintf("schema must be %s or %s", SchemaVersionV1, SchemaVersionV2), "update the rule schema value")
	}
	if rule.Schema == SchemaVersionV1 {
		if len(rule.Lenses) != 0 || rule.Directive != "" ||
			rule.Verification.Coverage != "" || rule.Verification.Proves != "" {
			add("schema", "rule v1 must not declare v2 activation or proof metadata", "remove lenses, directive, coverage, and proves or update schema to ssb.dev/rule/v2")
		}
	} else if rule.Schema == SchemaVersionV2 {
		diagnostics = append(diagnostics, validateRuleV2Activation(rule)...)
	}
	if !stableIDPattern.MatchString(rule.ID) {
		add("id", "id must be lower-case kebab-case", "choose a stable id such as verify-before-merge")
	}
	if fileName != rule.ID+".md" {
		add("id", fmt.Sprintf("filename %q must match rule id %q", fileName, rule.ID+".md"), "rename the source file or correct its id")
	}
	if strings.TrimSpace(rule.Title) == "" {
		add("title", "title is required", "add a concise developer-facing title")
	}
	if rule.Topic == "" {
		add("topic", "topic is required", topicRecovery)
	} else if _, supported := supportedTopics[rule.Topic]; !supported {
		add("topic", fmt.Sprintf("topic %q is not supported", rule.Topic), topicRecovery)
	}
	if strings.TrimSpace(rule.Body) == "" {
		add("body", "rule body is required", "write the exact guidance that should be projected")
	}
	if len(rule.Scopes) == 0 {
		add("scopes", "at least one path scope is required", "add one or more repository-relative glob scopes")
	}
	for _, scope := range rule.Scopes {
		if unsafeScope(scope) {
			add("scopes", fmt.Sprintf("unsafe or empty scope %q", scope), "use a repository-relative glob without path traversal")
		}
	}
	if rule.Classification != "guidance" && rule.Classification != "deterministic" {
		add("classification", "classification must be guidance or deterministic", "classify judgment separately from existing deterministic proof")
	}
	if rule.Confidence != "low" && rule.Confidence != "medium" && rule.Confidence != "high" {
		add("confidence", "confidence must be low, medium, or high", "record an honest confidence band")
	}
	if rule.BaselineCommit != repo.Baseline() {
		add("baseline_commit", fmt.Sprintf("baseline_commit must equal current HEAD %s", repo.Baseline()), "reinspect the new commit and refresh every evidence hash")
	}
	diagnostics = append(diagnostics, validateScore(rule)...)

	authoritative := false
	evidencePaths := make(map[string]struct{})
	evidenceLocations := make(map[string]struct{})
	uniqueEvidenceCount := 0
	if len(rule.Evidence) == 0 {
		add("evidence", "at least one evidence location is required", "cite exact baseline lines and their excerpt hash")
	}
	for index, evidence := range rule.Evidence {
		field := fmt.Sprintf("evidence[%d]", index)
		diagnostics = append(diagnostics, validateEvidence(ctx, repo, rule.SourcePath, field, evidence)...)
		authoritative = authoritative || evidence.Authoritative
		evidencePaths[evidence.Path] = struct{}{}
		location := evidence.Path + "\x00" + evidence.Lines
		if _, duplicate := evidenceLocations[location]; duplicate {
			add("evidence", fmt.Sprintf("duplicate evidence location %s:%s", evidence.Path, evidence.Lines), "cite each distinct occurrence once")
		} else {
			evidenceLocations[location] = struct{}{}
			uniqueEvidenceCount++
		}
	}
	if !authoritative && (uniqueEvidenceCount < 3 || len(evidencePaths) < 2) {
		add("evidence", "candidate requires one authoritative source or three occurrences across two files", "add consistent evidence or keep the candidate in assessment.md")
	}

	hasCommand := strings.TrimSpace(rule.Verification.Command) != ""
	hasSource := rule.Verification.Source != nil
	hasGap := strings.TrimSpace(rule.Verification.ProofGap) != ""
	hasCoverage := strings.TrimSpace(rule.Verification.Coverage) != ""
	hasProves := strings.TrimSpace(rule.Verification.Proves) != ""
	if rule.Classification == "deterministic" {
		if !hasCommand || !hasSource || hasGap {
			add("verification", "deterministic rules must cite an existing verification command and source without a proof gap", "cite the existing check or reclassify the rule as guidance")
		}
	} else if rule.Classification == "guidance" {
		if hasGap == (hasCommand || hasSource) {
			add("verification", "guidance must cite one existing check or one explicit proof gap", "set command plus source, or set proof_gap")
		}
		if hasCommand != hasSource {
			add("verification", "verification command and source must be provided together", "cite the repository location that defines the command")
		}
	}
	if rule.Schema == SchemaVersionV2 {
		switch {
		case hasGap:
			if hasCoverage || hasProves {
				add("verification", "proof gaps must not declare verification coverage or a proved property", "remove coverage and proves from the proof gap")
			}
		case hasCommand && hasSource:
			if !hasProves {
				add("verification.proves", "mapped verification must state the bounded property it proves", "describe only the property established when the cited command passes")
			}
			if rule.Classification == "deterministic" && rule.Verification.Coverage != "full" {
				add("verification.coverage", "deterministic rules require full verification coverage", "set coverage to full only when the cited check proves the complete rule")
			}
			if rule.Classification == "guidance" && rule.Verification.Coverage != "partial" {
				add("verification.coverage", "guidance with a mapped check requires partial verification coverage", "set coverage to partial or replace the mapping with a proof gap")
			}
		}
	}
	if hasSource {
		diagnostics = append(diagnostics, validateEvidence(ctx, repo, rule.SourcePath, "verification.source", *rule.Verification.Source)...)
	}

	seenSkills := make(map[string]struct{})
	for _, skillID := range rule.RelatedSkillIDs {
		if !stableIDPattern.MatchString(skillID) {
			add("related_skills", fmt.Sprintf("invalid related skill id %q", skillID), "use lower-case kebab-case skill directory names")
		}
		if _, duplicate := seenSkills[skillID]; duplicate {
			add("related_skills", fmt.Sprintf("duplicate related skill id %q", skillID), "list each related skill once")
		}
		seenSkills[skillID] = struct{}{}
	}
	return diagnostics
}

func validateRuleV2Activation(rule Rule) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(rule.SourcePath, field, message, recovery))
	}
	if len(rule.Lenses) == 0 {
		add("lenses", "at least one activation lens is required", "use one base lens or one or more language, framework, and task lenses")
	}
	baseCount := 0
	seen := make(map[string]struct{})
	for index, lens := range rule.Lenses {
		field := fmt.Sprintf("lenses[%d]", index)
		switch lens.Kind {
		case "base":
			baseCount++
			if lens.Value != "" {
				add(field+".value", "base lens must not have a value", "remove the value from the base lens")
			}
		case "language", "framework":
			if !stableIDPattern.MatchString(lens.Value) {
				add(field+".value", lens.Kind+" lens requires a lower-case kebab-case value", "add a value such as go, python, cobra, or django")
			}
		case "task":
			if _, supported := supportedTasks[lens.Value]; !supported {
				add(field+".value", fmt.Sprintf("task lens value %q is not supported", lens.Value), "use implementation, review, testing, security, documentation, or release")
			}
		default:
			add(field+".kind", fmt.Sprintf("lens kind %q is not supported", lens.Kind), "use base, language, framework, or task")
		}
		key := lens.Kind + ":" + lens.Value
		if _, duplicate := seen[key]; duplicate {
			add("lenses", "duplicate activation lens "+key, "list each lens once")
		}
		seen[key] = struct{}{}
	}
	if baseCount != 0 && len(rule.Lenses) != 1 {
		add("lenses", "base must be the sole activation lens", "remove contextual lenses or remove the base lens")
	}
	if _, supported := supportedDirectives[rule.Directive]; !supported {
		add("directive", fmt.Sprintf("directive %q is not supported", rule.Directive), "use always, ask-first, never, or prefer")
	}
	return diagnostics
}

func validateScore(rule Rule) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(rule.SourcePath, field, message, recovery))
	}
	if rule.Score.Method != ScoreMethod {
		add("score.method", fmt.Sprintf("score method must be %s", ScoreMethod), "use the versioned v0.1 ranking method")
	}
	factors := []struct {
		name  string
		value int
		max   int
	}{
		{"prevalence", rule.Score.Factors.Prevalence, 25},
		{"consistency", rule.Score.Factors.Consistency, 20},
		{"authority", rule.Score.Factors.Authority, 20},
		{"risk", rule.Score.Factors.Risk, 20},
		{"applicability", rule.Score.Factors.Applicability, 15},
	}
	sum := 0
	for _, factor := range factors {
		sum += factor.value
		if factor.value < 0 || factor.value > factor.max {
			add("score.factors."+factor.name, fmt.Sprintf("%s score %d is outside 0-%d", factor.name, factor.value, factor.max), "correct the factor within its documented range")
		}
	}
	if rule.Score.Total != sum {
		add("score.total", fmt.Sprintf("score total %d does not equal factor sum %d", rule.Score.Total, sum), "recalculate the transparent score")
	}
	expectedBand := importanceBand(rule.Score.Total)
	if expectedBand == "assessment-only" {
		add("score.total", fmt.Sprintf("score %d is below the 25-point rule threshold", rule.Score.Total), "keep this candidate in assessment.md instead of emitting a rule")
	} else if rule.Importance != expectedBand {
		add("importance", fmt.Sprintf("importance %s does not match score %d (%s)", rule.Importance, rule.Score.Total, expectedBand), "use the band determined by ssb-score-v1")
	}
	return diagnostics
}

func validateEvidence(ctx context.Context, repo *workspace.Repository, sourcePath, field string, evidence Evidence) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(sourcePath, field, message, recovery))
	}
	if evidence.Path == "" {
		add("evidence path is required", "cite a tracked regular file at the baseline commit")
		return diagnostics
	}
	if !digestPattern.MatchString(evidence.ExcerptSHA256) {
		add("excerpt_sha256 must use sha256:<64 lowercase hex characters>", "hash the exact cited bytes including line endings")
	}
	start, end, err := parseLineRange(evidence.Lines)
	if err != nil {
		add(err.Error(), "use a one-based range such as 12-18")
		return diagnostics
	}
	content, err := inventory.ReadEvidence(ctx, repo, evidence.Path)
	if err != nil {
		add(err.Error(), "cite an eligible tracked regular file listed by ssb inspect")
		return diagnostics
	}
	excerpt, err := lineExcerpt(content, start, end)
	if err != nil {
		add(err.Error(), "correct the cited line range")
		return diagnostics
	}
	sum := sha256.Sum256(excerpt)
	actual := "sha256:" + hex.EncodeToString(sum[:])
	if evidence.ExcerptSHA256 != actual {
		add(fmt.Sprintf("excerpt hash does not match %s lines %s; expected %s", evidence.Path, evidence.Lines, actual), "refresh the evidence after inspecting the exact baseline bytes")
	}
	return diagnostics
}

func loadSkill(root, skillID string) (Skill, []Diagnostic, error) {
	relative := path.Join(".agents/skills", skillID, "SKILL.md")
	if !stableIDPattern.MatchString(skillID) {
		return Skill{}, nil, nil
	}
	if component, found, err := findSymlinkComponent(root, relative); err != nil {
		return Skill{}, nil, err
	} else if found {
		return Skill{}, []Diagnostic{diagnostic(relative, "file", relative+" contains a symlink component "+component, "place the portable skill inside the repository")}, nil
	}
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	data, diagnostics, err := readRequiredRegularFile(absolute, relative)
	if err != nil || len(data) == 0 {
		return Skill{}, diagnostics, err
	}
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Skill{}, append(diagnostics, diagnostic(relative, "frontmatter", splitErr.Error(), "add Agent Skill YAML frontmatter")), nil
	}
	var metadata skillFrontmatter
	if err := yaml.Load(frontmatter, &metadata, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return Skill{}, append(diagnostics, diagnostic(relative, "frontmatter", err.Error(), "use only Agent Skills core specification fields")), nil
	}
	if metadata.Name != skillID {
		diagnostics = append(diagnostics, diagnostic(relative, "name", fmt.Sprintf("skill name %q must match directory %q", metadata.Name, skillID), "align the skill name and directory"))
	}
	if len(metadata.Name) > 64 {
		diagnostics = append(diagnostics, diagnostic(relative, "name", "skill name must be at most 64 characters", "shorten the portable skill name"))
	}
	if strings.TrimSpace(metadata.Description) == "" || len(metadata.Description) > 1024 {
		diagnostics = append(diagnostics, diagnostic(relative, "description", "skill description must contain 1-1024 characters", "describe what the skill does and when to use it"))
	}
	topic := metadata.Metadata["topic"]
	if topic == "" {
		diagnostics = append(diagnostics, diagnostic(relative, "metadata.topic", "topic is required", topicRecovery))
	} else if _, supported := supportedTopics[topic]; !supported {
		diagnostics = append(diagnostics, diagnostic(relative, "metadata.topic", fmt.Sprintf("topic %q is not supported", topic), topicRecovery))
	}
	if strings.TrimSpace(string(body)) == "" {
		diagnostics = append(diagnostics, diagnostic(relative, "body", "skill body is required", "document the procedural workflow"))
	}
	return Skill{
		ID:          skillID,
		Description: metadata.Description,
		Topic:       topic,
		SourcePath:  relative,
		Body:        string(body),
	}, diagnostics, nil
}

func findSymlinkComponent(root, relative string) (string, bool, error) {
	current := root
	for _, component := range strings.Split(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		if err != nil {
			return "", false, fmt.Errorf("inspect %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return filepath.ToSlash(current), true, nil
		}
	}
	return "", false, nil
}

func readRequiredRegularFile(absolute, relative string) ([]byte, []Diagnostic, error) {
	info, err := os.Lstat(absolute)
	if errors.Is(err, os.ErrNotExist) {
		return nil, []Diagnostic{diagnostic(relative, "file", relative+" does not exist", "create the required file and rerun ssb validate")}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("inspect %s: %w", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, []Diagnostic{diagnostic(relative, "file", relative+" must be a real regular file, not a symlink", "replace it with a file inside the repository")}, nil
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", relative, err)
	}
	if len(data) == 0 {
		return data, []Diagnostic{diagnostic(relative, "file", relative+" must not be empty", "write the required artifact and rerun ssb validate")}, nil
	}
	return data, nil, nil
}

func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	var opening []byte
	var closing []byte
	switch {
	case bytes.HasPrefix(data, []byte("---\n")):
		opening = []byte("---\n")
		closing = []byte("\n---\n")
	case bytes.HasPrefix(data, []byte("---\r\n")):
		opening = []byte("---\r\n")
		closing = []byte("\r\n---\r\n")
	default:
		return nil, nil, errors.New("file must begin with --- on its own line")
	}
	rest := data[len(opening):]
	index := bytes.Index(rest, closing)
	if index < 0 {
		return nil, nil, errors.New("frontmatter closing --- marker is missing")
	}
	frontmatter := rest[:index]
	body := rest[index+len(closing):]
	return frontmatter, body, nil
}

func parseLineRange(value string) (int, int, error) {
	parts := strings.Split(value, "-")
	if len(parts) < 1 || len(parts) > 2 {
		return 0, 0, fmt.Errorf("invalid evidence line range %q", value)
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil || start < 1 {
		return 0, 0, fmt.Errorf("invalid evidence line range %q", value)
	}
	end := start
	if len(parts) == 2 {
		end, err = strconv.Atoi(parts[1])
		if err != nil || end < start {
			return 0, 0, fmt.Errorf("invalid evidence line range %q", value)
		}
	}
	return start, end, nil
}

func lineExcerpt(content []byte, start, end int) ([]byte, error) {
	lines := splitLines(content)
	if start > len(lines) || end > len(lines) {
		return nil, fmt.Errorf("line range %d-%d exceeds file length %d", start, end, len(lines))
	}
	var excerpt []byte
	for _, line := range lines[start-1 : end] {
		excerpt = append(excerpt, line...)
	}
	return excerpt, nil
}

func splitLines(content []byte) [][]byte {
	if len(content) == 0 {
		return nil
	}
	lines := make([][]byte, 0, bytes.Count(content, []byte{'\n'})+1)
	for len(content) > 0 {
		index := bytes.IndexByte(content, '\n')
		if index < 0 {
			lines = append(lines, content)
			break
		}
		lines = append(lines, content[:index+1])
		content = content[index+1:]
	}
	return lines
}

func unsafeScope(scope string) bool {
	if strings.TrimSpace(scope) == "" || strings.HasPrefix(scope, "/") || strings.HasPrefix(scope, "\\") {
		return true
	}
	normalized := strings.ReplaceAll(scope, "\\", "/")
	if len(normalized) >= 3 && normalized[1] == ':' &&
		((normalized[0] >= 'A' && normalized[0] <= 'Z') || (normalized[0] >= 'a' && normalized[0] <= 'z')) &&
		normalized[2] == '/' {
		return true
	}
	for _, component := range strings.Split(normalized, "/") {
		if component == ".." {
			return true
		}
	}
	return false
}

func importanceBand(score int) string {
	switch {
	case score >= 80 && score <= 100:
		return "very-high"
	case score >= 65 && score <= 79:
		return "high"
	case score >= 45 && score <= 64:
		return "medium"
	case score >= 25 && score <= 44:
		return "low"
	default:
		return "assessment-only"
	}
}

func diagnostic(filePath, field, message, recovery string) Diagnostic {
	return Diagnostic{
		Path:     filePath,
		Field:    field,
		Message:  message,
		Recovery: recovery,
	}
}
