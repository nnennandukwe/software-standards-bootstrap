// Package adr creates one Proposed architecture decision record from the
// developer-retained semantic rules, verification recipes, and Agent Skills.
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
	ErrAmbiguousDirectory   = errors.New("multiple ADR directories found")
	ErrUnsafeTarget         = errors.New("unsafe ADR target")
	ErrCollision            = errors.New("ADR target already exists")
	ErrNoAdoptableArtifacts = errors.New("no adoptable artifacts")
	numberedADRPattern      = regexp.MustCompile(`^([0-9]+)-.+\.md$`)
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
	if len(pack.Rules) == 0 && len(pack.Recipes) == 0 && len(pack.Skills) == 0 {
		return Result{}, fmt.Errorf(
			"%w: retain at least one semantic rule, verification recipe, or Agent Skill before creating an ADR",
			ErrNoAdoptableArtifacts,
		)
	}
	relativeDir, absoluteDir, err := resolveDirectory(ctx, repo, options.Directory)
	if err != nil {
		return Result{}, err
	}
	number, width, err := nextNumber(absoluteDir)
	if err != nil {
		return Result{}, err
	}
	fileName := fmt.Sprintf("%0*d-actionable-standards.md", width, number)
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
	recipes := append([]rulepack.VerificationRecipe(nil), pack.Recipes...)
	skills := append([]rulepack.Skill(nil), pack.Skills...)
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	sort.Slice(recipes, func(i, j int) bool { return recipes[i].ID < recipes[j].ID })
	sort.Slice(skills, func(i, j int) bool { return skills[i].ID < skills[j].ID })
	manifest := make(map[string]rulepack.AcceptedArtifact, len(pack.Report.Artifacts))
	for _, artifact := range pack.Report.Artifacts {
		manifest[artifact.ID] = artifact
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# ADR %04d: Adopt actionable repository standards\n\n", number)
	output.WriteString("- Status: Proposed\n")
	fmt.Fprintf(&output, "- Baseline commit: `%s`\n", pack.BaselineCommit)
	if pack.Layout == rulepack.LayoutManifest {
		fmt.Fprintf(&output, "- Manifest: `%s`\n", pack.ManifestPath)
		fmt.Fprintf(&output, "- Inventory: `%s`\n", pack.InventoryPath)
		fmt.Fprintf(&output, "- Report: `%s`\n\n", pack.ReportPath)
	} else {
		fmt.Fprintf(&output, "- Report: `%s`\n\n", pack.ReportPath)
	}
	output.WriteString("## Context\n\n")
	output.WriteString("The repository was inspected at the pinned baseline above. The developer retained the following evidence-backed actionable artifacts after review. Verification recipes are recorded here but were not executed by SSB.\n")
	if len(rules) != 0 {
		output.WriteString("\n## Semantic rules\n")
	}
	for _, rule := range rules {
		metadata := manifest[rule.ID]
		fmt.Fprintf(&output, "\n### %s (`%s`)\n\n", rule.Title, rule.ID)
		fmt.Fprintf(&output, "- Source: `%s`\n", rule.SourcePath)
		fmt.Fprintf(&output, "- Scope: %s\n", markdownCodeList(rule.Scopes))
		fmt.Fprintf(&output, "- Lenses: %s\n", markdownLensList(rule.Lenses))
		fmt.Fprintf(&output, "- Directive: `%s`\n", rule.Directive)
		writeAdoptionMetadata(&output, rule.Category, rule.Derivation, rule.Evidence, metadata)
		output.WriteString("\n")
		output.WriteString(rule.Body)
		if !strings.HasSuffix(rule.Body, "\n") {
			output.WriteString("\n")
		}
	}
	if len(recipes) != 0 {
		output.WriteString("\n## Verification recipes\n")
		for _, recipe := range recipes {
			metadata := manifest[recipe.ID]
			fmt.Fprintf(&output, "\n### %s (`%s`)\n\n", recipe.Title, recipe.ID)
			fmt.Fprintf(&output, "- Source: `%s`\n", recipe.SourcePath)
			fmt.Fprintf(&output, "- Scope: %s\n", markdownCodeList(recipe.Scopes))
			fmt.Fprintf(&output, "- Lenses: %s\n", markdownLensList(recipe.Lenses))
			fmt.Fprintf(&output, "- When: %s\n", strings.TrimSpace(recipe.When))
			writeAdoptionMetadata(&output, recipe.Category, recipe.Derivation, recipe.Evidence, metadata)
		}
	}
	if len(skills) != 0 {
		output.WriteString("\n## Agent Skills\n")
		for _, skill := range skills {
			metadata := manifest[skill.ID]
			fmt.Fprintf(&output, "\n### %s (`%s`)\n\n", titleFromID(skill.ID), skill.ID)
			fmt.Fprintf(&output, "- Source: `%s`\n", skill.SourcePath)
			fmt.Fprintf(&output, "- Description: %s\n", strings.TrimSpace(skill.Description))
			fmt.Fprintf(&output, "- Scope: %s\n", markdownCodeList(metadata.Scopes))
			fmt.Fprintf(&output, "- Lenses: %s\n", markdownLensList(metadata.Lenses))
			writeAdoptionMetadata(&output, skill.Category, metadata.Derivation, metadata.Evidence, metadata)
		}
	}
	output.WriteString("\n## Consequences\n\n")
	if pack.Layout == rulepack.LayoutManifest {
		output.WriteString("- `AGENTS.md` is a derived projection; the manifest, inventory, human report, and canonical artifact source files remain editable.\n")
	} else {
		output.WriteString("- `AGENTS.md` is a derived projection; the report and canonical artifact source files remain editable.\n")
	}
	output.WriteString("- Verification recipes remain deliberately invoked repository procedures; this record does not claim their commands passed.\n")
	output.WriteString("- The developer-created pull request and its merge constitute adoption; this ADR remains Proposed until then.\n")
	return []byte(output.String())
}

func writeAdoptionMetadata(
	output *strings.Builder,
	category string,
	derivation string,
	evidence []rulepack.Evidence,
	metadata rulepack.AcceptedArtifact,
) {
	fmt.Fprintf(output, "- Category: `%s`\n", category)
	fmt.Fprintf(output, "- Derivation: `%s`\n", derivation)
	fmt.Fprintf(output, "- Confidence: `%s`\n", metadata.Confidence)
	fmt.Fprintf(
		output,
		"- Utility: `%s` (%d/100, `%s`)\n",
		utilityBand(metadata.Utility.Total),
		metadata.Utility.Total,
		metadata.Utility.Method,
	)
	fmt.Fprintf(output, "- Evidence: %s\n", markdownEvidenceList(evidence))
}

func markdownEvidenceList(evidence []rulepack.Evidence) string {
	values := make([]string, 0, len(evidence))
	for _, item := range evidence {
		values = append(
			values,
			fmt.Sprintf("`%s:%s` (`%s`)", item.Path, item.Lines, item.Role),
		)
	}
	return strings.Join(values, ", ")
}

func utilityBand(total int) string {
	switch {
	case total >= 80:
		return "very-high"
	case total >= 65:
		return "high"
	default:
		return "medium"
	}
}

func titleFromID(id string) string {
	title := strings.ReplaceAll(id, "-", " ")
	if title != "" {
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	return title
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
