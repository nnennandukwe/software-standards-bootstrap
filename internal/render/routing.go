package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
)

const (
	RoutingDirectory = ".software-standards/routing"
	routingStart     = "<!-- software-standards-bootstrap:routing:start -->"
	routingEnd       = "<!-- software-standards-bootstrap:routing:end -->"
)

func buildRoutingResult(pack rulepack.Pack) (*RoutingResult, error) {
	if pack.Routing == nil {
		return nil, fmt.Errorf("manifest pack has no normalized routing catalog")
	}
	rules := make(map[string]rulepack.Rule, len(pack.Rules))
	for _, rule := range pack.Rules {
		rules[rule.ID] = rule
	}
	recipes := make(map[string]rulepack.VerificationRecipe, len(pack.Recipes))
	for _, recipe := range pack.Recipes {
		recipes[recipe.ID] = recipe
	}
	skills := make(map[string]rulepack.Skill, len(pack.Skills))
	for _, skill := range pack.Skills {
		skills[skill.ID] = skill
	}
	artifacts := make(map[string]rulepack.AcceptedArtifact, len(pack.Manifest.Artifacts))
	for _, artifact := range pack.Manifest.Artifacts {
		artifacts[artifact.ID] = artifact
	}
	rootCore := make(map[string]struct{}, len(pack.Manifest.RootCore))
	for _, ruleID := range pack.Manifest.RootCore {
		rootCore[ruleID] = struct{}{}
	}

	bundleFiles := make([]FileResult, 0, len(pack.Routing.Bundles))
	for _, bundle := range pack.Routing.Bundles {
		body, err := buildBundleBody(bundle, artifacts, rules, recipes, skills, rootCore)
		if err != nil {
			return nil, err
		}
		source := struct {
			Bundle  rulepack.RoutingBundle        `json:"bundle"`
			Rules   []rulepack.Rule               `json:"rules"`
			Recipes []rulepack.VerificationRecipe `json:"recipes"`
			Skills  []rulepack.Skill              `json:"skills"`
			Index   []rulepack.AcceptedArtifact   `json:"index"`
			Root    []string                      `json:"root_core"`
		}{Bundle: bundle, Root: append([]string(nil), pack.Manifest.RootCore...)}
		for _, artifactID := range bundle.ArtifactIDs {
			if artifact, exists := artifacts[artifactID]; exists {
				source.Index = append(source.Index, artifact)
			}
			if rule, exists := rules[artifactID]; exists {
				source.Rules = append(source.Rules, rule)
			}
			if recipe, exists := recipes[artifactID]; exists {
				source.Recipes = append(source.Recipes, recipe)
			}
			if skill, exists := skills[artifactID]; exists {
				source.Skills = append(source.Skills, skill)
			}
		}
		content, err := wrapRoutingFile(source, []byte(body))
		if err != nil {
			return nil, err
		}
		bundleFiles = append(bundleFiles, FileResult{
			Path: bundle.Path, SHA256: digest(content), Bytes: len(content), Content: content,
		})
	}
	sort.Slice(bundleFiles, func(i, j int) bool { return bundleFiles[i].Path < bundleFiles[j].Path })

	catalogBody := buildCatalogBody(pack, bundleFiles, artifacts, rules, recipes, skills)
	catalogSource := struct {
		Routing     rulepack.RoutingCatalog        `json:"routing"`
		BundleFiles []fileIdentity                 `json:"bundle_files"`
		Guidance    []rulepack.OrientationGuidance `json:"task_guidance,omitempty"`
	}{Routing: *pack.Routing, BundleFiles: fileIdentities(bundleFiles)}
	if pack.Orientation != nil {
		for _, guidance := range pack.Orientation.Guidance {
			if guidance.Kind == "planning" || guidance.Kind == "implementation" {
				catalogSource.Guidance = append(catalogSource.Guidance, guidance)
			}
		}
	}
	catalogContent, err := wrapRoutingFile(catalogSource, []byte(catalogBody))
	if err != nil {
		return nil, err
	}
	catalog := FileResult{
		Path: pack.Routing.CatalogPath, SHA256: digest(catalogContent), Bytes: len(catalogContent), Content: catalogContent,
	}
	files := append([]FileResult{catalog}, bundleFiles...)
	return &RoutingResult{
		Path: RoutingDirectory, Changed: true, Exists: true,
		TreeDigest: routingTreeDigest(files), Files: files,
	}, nil
}

type fileIdentity struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func fileIdentities(files []FileResult) []fileIdentity {
	result := make([]fileIdentity, 0, len(files))
	for _, file := range files {
		result = append(result, fileIdentity{Path: file.Path, SHA256: file.SHA256})
	}
	return result
}

func routingTreeDigest(files []FileResult) string {
	identities := fileIdentities(files)
	sort.Slice(identities, func(i, j int) bool { return identities[i].Path < identities[j].Path })
	encoded, err := json.Marshal(identities)
	if err != nil {
		panic(fmt.Sprintf("encode generated routing tree: %v", err))
	}
	return digest(encoded)
}

func wrapRoutingFile(source any, body []byte) ([]byte, error) {
	canonical, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("encode routing source digest: %w", err)
	}
	if bytes.Contains(body, []byte(routingStart)) || bytes.Contains(body, []byte(routingEnd)) {
		return nil, fmt.Errorf("%w: routing projection contains a reserved marker", ErrMarkers)
	}
	sourceLine := "<!-- source-digest: " + digest(canonical) + " -->"
	contentDigest := digest(append([]byte(sourceLine+"\n"), body...))
	var result strings.Builder
	result.WriteString(routingStart)
	result.WriteByte('\n')
	result.WriteString(sourceLine)
	result.WriteByte('\n')
	fmt.Fprintf(&result, "<!-- content-digest: %s -->\n", contentDigest)
	result.Write(body)
	result.WriteString(routingEnd)
	result.WriteByte('\n')
	return []byte(result.String()), nil
}

func buildCatalogBody(
	pack rulepack.Pack,
	files []FileResult,
	artifacts map[string]rulepack.AcceptedArtifact,
	rules map[string]rulepack.Rule,
	recipes map[string]rulepack.VerificationRecipe,
	skills map[string]rulepack.Skill,
) string {
	fileByPath := make(map[string]FileResult, len(files))
	for _, file := range files {
		fileByPath[file.Path] = file
	}
	var body strings.Builder
	body.WriteString("# Software Standards Bootstrap routing catalog\n\n")
	body.WriteString("This generated catalog routes a task to portable Markdown bundles. Canonical artifact sources and manifest metadata remain authoritative. File presence does not prove adoption, command execution, verification, or release.\n\n")
	body.WriteString("The root routing-tree digest is SHA-256 over compact JSON for the path-sorted list of `{\"path\":<repository-relative path>,\"sha256\":<sha256:file-digest>}` identities containing this catalog and every bundle.\n\n")
	body.WriteString("## Select bundles\n\n")
	body.WriteString("1. Identify every affected repository-relative path and classify the task as `planning`, `implementation`, or `verification`.\n")
	body.WriteString("2. Identify languages and frameworks only from the request and repository evidence already available.\n")
	body.WriteString("3. Select a bundle when at least one scope matches and every represented lens dimension matches; values inside one dimension are alternatives. A `base` lens matches every task.\n")
	body.WriteString("4. If a path, language, framework, or task is uncertain, include potentially relevant bundles instead of excluding them.\n")
	body.WriteString("5. Read the selected bundle and every linked Agent Skill before acting. Recipe commands remain inert until separately authorized and executed.\n")
	if pack.Orientation != nil {
		wrote := false
		for _, guidance := range pack.Orientation.Guidance {
			if guidance.Kind != "planning" && guidance.Kind != "implementation" {
				continue
			}
			if !wrote {
				body.WriteString("\n## Task guidance\n")
				wrote = true
			}
			fmt.Fprintf(&body, "\n- **%s:** %s\n", titleFromID(guidance.Kind), markdownText(guidance.Text))
		}
	}
	body.WriteString("\n## Bundles\n")
	for _, bundle := range pack.Routing.Bundles {
		file := fileByPath[bundle.Path]
		fmt.Fprintf(&body, "\n### %s\n\n", linkFrom(rulepack.RoutingCatalogPath, bundle.ID, bundle.Path))
		fmt.Fprintf(&body, "- Scopes: %s\n", codeList(bundle.Scopes))
		fmt.Fprintf(&body, "- Lenses: %s\n", lensCodeList(bundle.Lenses))
		fmt.Fprintf(&body, "- SHA-256: %s\n", inlineCode(file.SHA256))
		body.WriteString("- Artifacts:\n")
		for _, artifactID := range bundle.ArtifactIDs {
			artifact := artifacts[artifactID]
			title := titleFromID(artifactID)
			if rule, exists := rules[artifactID]; exists {
				title = rule.Title
			} else if recipe, exists := recipes[artifactID]; exists {
				title = recipe.Title
			} else if _, exists := skills[artifactID]; exists {
				title = titleFromID(artifactID)
			}
			fmt.Fprintf(&body, "  - %s %s (%s)\n", titleFromID(artifact.Kind), linkFrom(rulepack.RoutingCatalogPath, title, artifact.Path), inlineCode(artifactID))
		}
	}
	return body.String()
}

func buildBundleBody(
	bundle rulepack.RoutingBundle,
	artifacts map[string]rulepack.AcceptedArtifact,
	rules map[string]rulepack.Rule,
	recipes map[string]rulepack.VerificationRecipe,
	skills map[string]rulepack.Skill,
	rootCore map[string]struct{},
) (string, error) {
	var body strings.Builder
	fmt.Fprintf(&body, "# Routing bundle %s\n\n", inlineCode(bundle.ID))
	fmt.Fprintf(&body, "- Scopes: %s\n", codeList(bundle.Scopes))
	fmt.Fprintf(&body, "- Lenses: %s\n", lensCodeList(bundle.Lenses))
	body.WriteString("- Commands shown below are inert declarations, not execution evidence.\n")

	wroteRules := false
	for _, artifactID := range bundle.ArtifactIDs {
		rule, exists := rules[artifactID]
		if !exists {
			continue
		}
		if !wroteRules {
			body.WriteString("\n## Semantic rules\n")
			wroteRules = true
		}
		fmt.Fprintf(&body, "\n### %s (%s) - %s\n\n", linkFrom(bundle.Path, rule.Title, rule.SourcePath), inlineCode(rule.ID), inlineCode(rule.Directive))
		if _, projected := rootCore[rule.ID]; projected {
			body.WriteString("This rule is projected in root `AGENTS.md`; use the canonical link above for its reviewed source.\n")
			continue
		}
		writeQuotedMarkdown(&body, rule.Body)
		if !strings.HasSuffix(rule.Body, "\n") {
			body.WriteByte('\n')
		}
	}

	wroteRecipes := false
	for _, artifactID := range bundle.ArtifactIDs {
		recipe, exists := recipes[artifactID]
		if !exists {
			continue
		}
		if !wroteRecipes {
			body.WriteString("\n## Verification commands\n")
			wroteRecipes = true
		}
		fmt.Fprintf(&body, "\n### %s (%s)\n\n", linkFrom(bundle.Path, recipe.Title, recipe.SourcePath), inlineCode(recipe.ID))
		fmt.Fprintf(&body, "When: %s\n", markdownText(strings.TrimSpace(recipe.When)))
		for index, step := range recipe.Steps {
			fmt.Fprintf(&body, "\n#### Step %d\n\n", index+1)
			if step.WorkingDirectory != "." {
				fmt.Fprintf(&body, "Working directory: %s\n\n", inlineCode(step.WorkingDirectory))
			}
			writeCommandFence(&body, step.Run)
			fmt.Fprintf(&body, "\nExpected result: %s\n", markdownText(step.ExpectedResult))
		}
	}

	wroteSkills := false
	for _, artifactID := range bundle.ArtifactIDs {
		skill, exists := skills[artifactID]
		if !exists {
			continue
		}
		if !wroteSkills {
			body.WriteString("\n## Agent Skills\n")
			wroteSkills = true
		}
		fmt.Fprintf(&body, "\n- %s (%s) - %s Read the complete skill before using it.\n", linkFrom(bundle.Path, titleFromID(skill.ID), skill.SourcePath), inlineCode(skill.ID), markdownText(strings.TrimSpace(skill.Description)))
	}
	for _, artifactID := range bundle.ArtifactIDs {
		if _, exists := artifacts[artifactID]; !exists {
			return "", fmt.Errorf("routing bundle %s references unknown artifact %s", bundle.ID, artifactID)
		}
	}
	return body.String(), nil
}

func linkFrom(fromFile, label, target string) string {
	return markdownLink(label, relativePath(path.Dir(fromFile), target))
}

func relativePath(fromDirectory, target string) string {
	from := strings.Split(path.Clean(fromDirectory), "/")
	to := strings.Split(path.Clean(target), "/")
	common := 0
	for common < len(from) && common < len(to) && from[common] == to[common] {
		common++
	}
	parts := make([]string, 0, len(from)-common+len(to)-common)
	for index := common; index < len(from); index++ {
		parts = append(parts, "..")
	}
	parts = append(parts, to[common:]...)
	if len(parts) == 0 {
		return "."
	}
	return path.Join(parts...)
}
