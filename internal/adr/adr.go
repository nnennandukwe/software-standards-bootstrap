// Package adr creates one Proposed architecture decision record from the
// developer-retained rule and skill source files.
package adr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

var (
	ErrAmbiguousDirectory = errors.New("multiple ADR directories found")
	ErrUnsafeTarget       = errors.New("unsafe ADR target")
	ErrCollision          = errors.New("ADR target already exists")
	numberedADRPattern    = regexp.MustCompile(`^([0-9]+)-.+\.md$`)
)

// Options controls directory selection and dry-run behavior.
type Options struct {
	Directory string
	DryRun    bool
}

// Result describes the one new ADR.
type Result struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
	DryRun  bool   `json:"dry_run"`
	Content []byte `json:"-"`
}

// Create chooses the repository's existing ADR convention or defaults to
// docs/adr. It uses exclusive creation and never overwrites an existing file.
func Create(ctx context.Context, repo *workspace.Repository, pack rulepack.Pack, options Options) (Result, error) {
	relativeDir, absoluteDir, err := resolveDirectory(ctx, repo, options.Directory)
	if err != nil {
		return Result{}, err
	}
	number, width, err := nextNumber(absoluteDir)
	if err != nil {
		return Result{}, err
	}
	fileName := fmt.Sprintf("%0*d-agentic-rules.md", width, number)
	relativePath := filepath.ToSlash(filepath.Join(relativeDir, fileName))
	if relativeDir == "." {
		relativePath = fileName
	}
	absolutePath := filepath.Join(absoluteDir, fileName)
	if _, err := os.Lstat(absolutePath); err == nil {
		return Result{}, fmt.Errorf("%w: %s", ErrCollision, relativePath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, fmt.Errorf("inspect ADR target: %w", err)
	}

	content := render(pack, number)
	result := Result{
		Path:    relativePath,
		Created: false,
		DryRun:  options.DryRun,
		Content: content,
	}
	if options.DryRun {
		return result, nil
	}

	if err := os.MkdirAll(absoluteDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create ADR directory: %w", err)
	}
	if err := checkDirectoryPath(repo.Root(), absoluteDir); err != nil {
		return Result{}, err
	}
	file, err := os.OpenFile(absolutePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return Result{}, fmt.Errorf("%w: %s", ErrCollision, relativePath)
	}
	if err != nil {
		return Result{}, fmt.Errorf("create ADR %s: %w", relativePath, err)
	}
	created := true
	defer func() {
		_ = file.Close()
		if created {
			_ = os.Remove(absolutePath)
		}
	}()
	if _, err := file.Write(content); err != nil {
		return Result{}, fmt.Errorf("write ADR %s: %w", relativePath, err)
	}
	if err := file.Sync(); err != nil {
		return Result{}, fmt.Errorf("sync ADR %s: %w", relativePath, err)
	}
	if err := file.Close(); err != nil {
		return Result{}, fmt.Errorf("close ADR %s: %w", relativePath, err)
	}
	created = false
	result.Created = true
	return result, nil
}

func resolveDirectory(ctx context.Context, repo *workspace.Repository, requested string) (string, string, error) {
	if requested != "" {
		relative, absolute, err := repositoryDirectory(repo.Root(), requested)
		if err != nil {
			return "", "", err
		}
		if err := checkDirectoryPath(repo.Root(), absolute); err != nil {
			return "", "", err
		}
		if relative != "." {
			submodule, err := repo.HasSubmodulePrefix(ctx, filepath.ToSlash(relative))
			if err != nil {
				return "", "", err
			}
			if submodule {
				return "", "", fmt.Errorf("%w: ADR directory is inside a submodule", ErrUnsafeTarget)
			}
		}
		return relative, absolute, nil
	}

	candidates := []string{"docs/adr", "docs/adrs", "adr", "adrs"}
	existing := make([]string, 0)
	for _, candidate := range candidates {
		absolute := filepath.Join(repo.Root(), filepath.FromSlash(candidate))
		info, err := os.Lstat(absolute)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", "", fmt.Errorf("inspect ADR convention %s: %w", candidate, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", "", fmt.Errorf("%w: ADR convention path %s must be a real directory", ErrUnsafeTarget, candidate)
		}
		existing = append(existing, candidate)
	}
	if len(existing) > 1 {
		return "", "", fmt.Errorf("%w (%s); pass --adr-dir PATH", ErrAmbiguousDirectory, strings.Join(existing, ", "))
	}
	relative := "docs/adr"
	if len(existing) == 1 {
		relative = existing[0]
	}
	absolute := filepath.Join(repo.Root(), filepath.FromSlash(relative))
	if err := checkDirectoryPath(repo.Root(), absolute); err != nil {
		return "", "", err
	}
	submodule, err := repo.HasSubmodulePrefix(ctx, relative)
	if err != nil {
		return "", "", err
	}
	if submodule {
		return "", "", fmt.Errorf("%w: ADR directory is inside a submodule", ErrUnsafeTarget)
	}
	return relative, absolute, nil
}

func repositoryDirectory(root, requested string) (string, string, error) {
	absolute := requested
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, requested)
	}
	absolute = filepath.Clean(absolute)
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return "", "", fmt.Errorf("%w: resolve --adr-dir: %v", ErrUnsafeTarget, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("%w: --adr-dir must stay inside the repository", ErrUnsafeTarget)
	}
	return relative, absolute, nil
}

func checkDirectoryPath(root, directory string) error {
	relative, err := filepath.Rel(root, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: ADR directory must stay inside the repository", ErrUnsafeTarget)
	}
	current := root
	if relative == "." {
		return nil
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect ADR directory component: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: ADR directory contains symlink component %s", ErrUnsafeTarget, current)
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: ADR directory component %s is not a directory", ErrUnsafeTarget, current)
		}
	}
	return nil
}

func nextNumber(directory string) (int, int, error) {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return 1, 4, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("read ADR directory: %w", err)
	}
	maxNumber := 0
	width := 4
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		match := numberedADRPattern.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		if number > maxNumber {
			maxNumber = number
		}
		if len(match[1]) > width {
			width = len(match[1])
		}
	}
	return maxNumber + 1, width, nil
}

func render(pack rulepack.Pack, number int) []byte {
	rules := append([]rulepack.Rule(nil), pack.Rules...)
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	referencedSkills := make(map[string]struct{})
	for _, rule := range rules {
		for _, skillID := range rule.RelatedSkillIDs {
			referencedSkills[skillID] = struct{}{}
		}
	}
	skills := make([]rulepack.Skill, 0)
	for _, skill := range pack.Skills {
		if _, retained := referencedSkills[skill.ID]; retained {
			skills = append(skills, skill)
		}
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].ID < skills[j].ID })

	var output strings.Builder
	fmt.Fprintf(&output, "# ADR %04d: Adopt agentic repository rules\n\n", number)
	output.WriteString("- Status: Proposed\n")
	fmt.Fprintf(&output, "- Baseline commit: `%s`\n", pack.BaselineCommit)
	fmt.Fprintf(&output, "- Assessment: `%s`\n\n", pack.AssessmentPath)
	output.WriteString("## Context\n\n")
	output.WriteString("The repository was inspected at the pinned baseline above. The developer retained the following evidence-backed source files after review. This record does not claim that cited verification commands were executed.\n\n")
	output.WriteString("## Proposed decision\n")
	for _, rule := range rules {
		fmt.Fprintf(&output, "\n### %s (`%s`)\n\n", rule.Title, rule.ID)
		fmt.Fprintf(&output, "- Source: `%s`\n", rule.SourcePath)
		fmt.Fprintf(&output, "- Scope: %s\n", markdownCodeList(rule.Scopes))
		fmt.Fprintf(&output, "- Primary topic: `%s`\n", rule.Topic)
		if rule.Schema == rulepack.SchemaVersionV2 {
			fmt.Fprintf(&output, "- Lenses: %s\n", markdownLensList(rule.Lenses))
			fmt.Fprintf(&output, "- Directive: `%s`\n", rule.Directive)
		}
		fmt.Fprintf(&output, "- Classification: `%s`\n", rule.Classification)
		fmt.Fprintf(&output, "- Importance: `%s` (%d/100, `%s`)\n", rule.Importance, rule.Score.Total, rule.Score.Method)
		fmt.Fprintf(&output, "- Confidence: `%s`\n", rule.Confidence)
		command := strings.TrimSpace(rule.Verification.Command)
		if command != "" {
			fmt.Fprintf(&output, "- Existing verification: `%s` (mapped, not executed)\n", command)
			if rule.Schema == rulepack.SchemaVersionV2 {
				fmt.Fprintf(&output, "- Verification coverage: `%s`\n", rule.Verification.Coverage)
				fmt.Fprintf(&output, "- Proves when the mapped command passes: %s\n", strings.TrimSpace(rule.Verification.Proves))
			}
		} else {
			fmt.Fprintf(&output, "- Proof gap: %s\n", strings.TrimSpace(rule.Verification.ProofGap))
		}
		output.WriteString("- Evidence:\n")
		for _, evidence := range rule.Evidence {
			fmt.Fprintf(&output, "  - `%s:%s` (`%s`)\n", evidence.Path, evidence.Lines, evidence.ExcerptSHA256)
		}
		if len(rule.RelatedSkillIDs) != 0 {
			fmt.Fprintf(&output, "- Related skills: %s\n", markdownCodeList(rule.RelatedSkillIDs))
		}
		output.WriteString("\n")
		output.WriteString(rule.Body)
		if !strings.HasSuffix(rule.Body, "\n") {
			output.WriteString("\n")
		}
	}
	if len(skills) != 0 {
		output.WriteString("\n## Retained procedural skills\n")
		for _, skill := range skills {
			fmt.Fprintf(&output, "\n- `%s`\n", skill.ID)
			fmt.Fprintf(&output, "  - Primary topic: `%s`\n", skill.Topic)
			fmt.Fprintf(&output, "  - Source: `%s`\n", skill.SourcePath)
			fmt.Fprintf(&output, "  - Description: %s\n", skill.Description)
		}
	}
	output.WriteString("\n## Consequences\n\n")
	output.WriteString("- `AGENTS.md` is a derived projection; rule and skill source files remain editable.\n")
	output.WriteString("- Guidance remains distinct from deterministic proof, and cited checks remain repository-owned.\n")
	output.WriteString("- The developer-created pull request and its merge constitute adoption; this ADR remains Proposed until then.\n")
	return []byte(output.String())
}

func markdownCodeList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}

func markdownLensList(lenses []rulepack.Lens) string {
	values := make([]string, 0, len(lenses))
	for _, lens := range lenses {
		value := lens.Kind
		if lens.Value != "" {
			value += ":" + lens.Value
		}
		values = append(values, value)
	}
	return markdownCodeList(values)
}
