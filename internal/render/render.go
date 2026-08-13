// Package render projects validated actionable artifacts into one bounded
// AGENTS.md managed section while preserving all surrounding bytes.
package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

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
	Exists        bool   `json:"exists"`
	SourceDigest  string `json:"source_digest"`
	ContentDigest string `json:"content_digest"`
	OutputDigest  string `json:"output_digest"`
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
	if len(pack.Rules) == 0 && len(pack.Recipes) == 0 && len(pack.Skills) == 0 {
		next, err := removeManagedSection(existing)
		if err != nil {
			if pack.Layout == rulepack.LayoutManifest && errors.Is(err, ErrDrift) {
				return Result{}, fmt.Errorf("%w: edit digest-bound sources and update manifest.yaml SHA-256 values instead of editing the generated section", ErrDrift)
			}
			return Result{}, err
		}
		result := Result{
			Path:          "AGENTS.md",
			Changed:       !bytes.Equal(existing, next),
			DryRun:        dryRun,
			Exists:        exists,
			SourceDigest:  digest(nil),
			ContentDigest: digest(nil),
			OutputDigest:  digest(next),
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

	section, sourceDigest, contentDigest, err := buildSection(pack)
	if err != nil {
		return Result{}, err
	}
	next, err := replaceManagedSection(existing, section)
	if err != nil {
		if pack.Layout == rulepack.LayoutManifest && errors.Is(err, ErrDrift) {
			return Result{}, fmt.Errorf("%w: edit digest-bound sources and update manifest.yaml SHA-256 values instead of editing the generated section", ErrDrift)
		}
		return Result{}, err
	}
	result := Result{
		Path:          "AGENTS.md",
		Changed:       !bytes.Equal(existing, next),
		DryRun:        dryRun,
		Exists:        true,
		SourceDigest:  sourceDigest,
		ContentDigest: contentDigest,
		OutputDigest:  digest(next),
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
	recipes := append([]rulepack.VerificationRecipe(nil), pack.Recipes...)
	skills := append([]rulepack.Skill(nil), pack.Skills...)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})
	sort.Slice(recipes, func(i, j int) bool { return recipes[i].ID < recipes[j].ID })
	sort.Slice(skills, func(i, j int) bool { return skills[i].ID < skills[j].ID })

	manifest := make(map[string]rulepack.AcceptedArtifact, len(pack.Report.Artifacts))
	for _, artifact := range pack.Report.Artifacts {
		manifest[artifact.ID] = artifact
	}
	sourceState := struct {
		Baseline        string                        `json:"baseline"`
		Manifest        []rulepack.AcceptedArtifact   `json:"manifest"`
		Rules           []rulepack.Rule               `json:"rules"`
		Recipes         []rulepack.VerificationRecipe `json:"recipes"`
		Skills          []rulepack.Skill              `json:"skills"`
		Layout          rulepack.Layout               `json:"layout,omitempty"`
		ManifestPath    string                        `json:"manifest_path,omitempty"`
		ManifestSchema  string                        `json:"manifest_schema,omitempty"`
		InventoryFile   *rulepack.FileReference       `json:"inventory_file,omitempty"`
		ReportFile      *rulepack.FileReference       `json:"report_file,omitempty"`
		OrientationFile *rulepack.FileReference       `json:"orientation_file,omitempty"`
		Orientation     *rulepack.Orientation         `json:"orientation,omitempty"`
	}{
		Baseline: pack.BaselineCommit,
		Manifest: renderableManifest(pack.Report.Artifacts),
		Rules:    rules,
		Recipes:  recipes,
		Skills:   skills,
	}
	if pack.Layout == rulepack.LayoutManifest {
		sourceState.Layout = pack.Layout
		sourceState.ManifestPath = pack.ManifestPath
		sourceState.ManifestSchema = pack.Manifest.Schema
		sourceState.InventoryFile = &pack.Manifest.Inventory
		sourceState.ReportFile = &pack.Manifest.Report
		if pack.Orientation != nil {
			sourceState.OrientationFile = &pack.Manifest.Orientation
			sourceState.Orientation = pack.Orientation
		}
	}
	var canonicalJSON bytes.Buffer
	encoder := json.NewEncoder(&canonicalJSON)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(sourceState); err != nil {
		return nil, "", "", fmt.Errorf("encode projection source digest: %w", err)
	}
	canonical := bytes.TrimSuffix(canonicalJSON.Bytes(), []byte("\n"))
	sourceDigest := digest(canonical)
	if bytes.Contains(canonical, []byte(StartMarker)) || bytes.Contains(canonical, []byte(EndMarker)) {
		return nil, "", "", fmt.Errorf("%w: canonical projection input contains a reserved marker", ErrMarkers)
	}
	orderedRules := append([]rulepack.Rule(nil), rules...)
	sort.SliceStable(orderedRules, func(i, j int) bool {
		leftDirective := directiveRank(orderedRules[i].Directive)
		rightDirective := directiveRank(orderedRules[j].Directive)
		if leftDirective != rightDirective {
			return leftDirective < rightDirective
		}
		leftUtility := manifest[orderedRules[i].ID].Utility.Total
		rightUtility := manifest[orderedRules[j].ID].Utility.Total
		if leftUtility != rightUtility {
			return leftUtility > rightUtility
		}
		return orderedRules[i].ID < orderedRules[j].ID
	})
	titles := artifactTitles(rules, recipes, skills)

	var body strings.Builder
	body.WriteString("## Software Standards Bootstrap\n\n")
	body.WriteString("This managed section is derived from retained canonical sources. An unmerged generated change is a proposal; repository review and merge are the adoption decision. File presence alone does not prove adoption.\n\n")
	if pack.Layout == rulepack.LayoutManifest {
		sources := []string{pack.ManifestPath, pack.InventoryPath, pack.ReportPath}
		if pack.Orientation != nil {
			sources = append(sources, pack.OrientationPath)
		}
		fmt.Fprintf(&body, "Generated from %s and the accepted artifacts by `ssb render`. Edit canonical sources and the manifest together, then rerun the command.\n\n", codeList(sources))
	} else {
		body.WriteString("Generated from `.software-standards/report.md` and its accepted artifacts by `ssb render`. Edit canonical sources and the report index together, then rerun the command.\n\n")
	}
	fmt.Fprintf(&body, "Baseline: `%s`\n\n", pack.BaselineCommit)
	body.WriteString("SSB did not stage, commit, push, open a pull request, execute any displayed recipe command, or activate another system. Recipe presence and expected results are not execution evidence.\n")

	if pack.Layout == rulepack.LayoutManifest && hasOrientationContent(pack.Orientation) {
		writeOrientation(&body, pack.Orientation, manifest, titles)
	}
	body.WriteString("\n### How routing works\n\n")
	body.WriteString("- Directory placement and nearest-file precedence are host-level `AGENTS.md` behavior.\n")
	body.WriteString("- Scopes and lenses are SSB's agent-readable routing contract, not native `AGENTS.md` glob activation. A semantic rule applies when its affected path scope matches; contextual artifacts also require every represented lens dimension to match, with values inside one dimension treated as alternatives.\n")
	body.WriteString("- If the language, framework, task, or affected path is uncertain, load the potentially relevant rule, recipe, or skill instead of excluding it.\n")
	body.WriteString("- Directives mean: `never` is prohibited, `ask-first` requires developer authorization, `always` is required, and `prefer` is the default when no documented exception or explicit user direction applies.\n")
	body.WriteString("- Linked artifact files are canonical. This projection is a concise router, not a replacement for their complete content.\n")

	writeStandingOrders(&body, orderedRules, manifest, titles)
	writeContextualRules(&body, orderedRules, manifest, titles)
	writeVerificationRecipes(&body, recipes, manifest, titles)
	writeSkills(&body, skills, manifest, titles)

	bodyBytes := []byte(body.String())
	if bytes.Contains(bodyBytes, []byte(StartMarker)) || bytes.Contains(bodyBytes, []byte(EndMarker)) {
		return nil, "", "", fmt.Errorf("%w: projection contains a reserved marker", ErrMarkers)
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

func renderableManifest(artifacts []rulepack.AcceptedArtifact) []rulepack.AcceptedArtifact {
	result := make([]rulepack.AcceptedArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Kind != "automation" {
			result = append(result, artifact)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func artifactTitles(
	rules []rulepack.Rule,
	recipes []rulepack.VerificationRecipe,
	skills []rulepack.Skill,
) map[string]string {
	titles := make(map[string]string, len(rules)+len(recipes)+len(skills))
	for _, rule := range rules {
		titles[rule.ID] = rule.Title
	}
	for _, recipe := range recipes {
		titles[recipe.ID] = recipe.Title
	}
	for _, skill := range skills {
		titles[skill.ID] = titleFromID(skill.ID)
	}
	return titles
}

func hasOrientationContent(orientation *rulepack.Orientation) bool {
	return orientation != nil && (orientation.Summary != nil || len(orientation.Areas) != 0 ||
		len(orientation.Prerequisites) != 0 || len(orientation.Documents) != 0 ||
		len(orientation.RelatedArtifactIDs) != 0 || len(orientation.Guidance) != 0)
}

func writeOrientation(
	body *strings.Builder,
	orientation *rulepack.Orientation,
	manifest map[string]rulepack.AcceptedArtifact,
	titles map[string]string,
) {
	body.WriteString("\n### Repository orientation\n")
	if orientation.Summary != nil {
		fmt.Fprintf(body, "\n%s\n\n- Evidence: %s\n", markdownText(orientation.Summary.Text), evidenceSourceList(orientation.Summary.Evidence, true))
	}
	if len(orientation.Areas) != 0 {
		body.WriteString("\n#### Important areas\n")
		for _, area := range orientation.Areas {
			fmt.Fprintf(body, "\n- %s — %s\n", inlineCode(area.Path), markdownText(area.Purpose))
			fmt.Fprintf(body, "  - Evidence: %s\n", evidenceSourceList(area.Evidence, true))
		}
	}
	if len(orientation.Prerequisites) != 0 {
		body.WriteString("\n#### Prerequisites\n")
		for _, prerequisite := range orientation.Prerequisites {
			fmt.Fprintf(body, "\n- %s\n", markdownText(prerequisite.Requirement))
			fmt.Fprintf(body, "  - Evidence: %s\n", evidenceSourceList(prerequisite.Evidence, true))
		}
	}
	if len(orientation.Documents) != 0 {
		body.WriteString("\n#### Canonical documents\n")
		for _, document := range orientation.Documents {
			fmt.Fprintf(body, "\n- %s\n", markdownLink(document.Label, document.Path))
			fmt.Fprintf(body, "  - Evidence: %s\n", evidenceSourceList(document.Evidence, true))
		}
	}
	if len(orientation.RelatedArtifactIDs) != 0 {
		body.WriteString("\n#### Related standards\n\n")
		for _, relatedID := range orientation.RelatedArtifactIDs {
			writeRelatedArtifact(body, relatedID, manifest, titles, "")
		}
	}
	if len(orientation.Guidance) != 0 {
		body.WriteString("\n#### Task guidance\n")
		for _, guidance := range orientation.Guidance {
			fmt.Fprintf(body, "\n- **%s:** %s\n", titleFromID(guidance.Kind), markdownText(guidance.Text))
			fmt.Fprintf(body, "  - Evidence: %s\n", evidenceSourceList(guidance.Evidence, true))
		}
	}
}

func writeStandingOrders(
	body *strings.Builder,
	rules []rulepack.Rule,
	manifest map[string]rulepack.AcceptedArtifact,
	titles map[string]string,
) {
	body.WriteString("\n### Standing orders\n")
	wroteAny := false
	for _, directive := range []string{"never", "ask-first", "always", "prefer"} {
		group := make([]rulepack.Rule, 0)
		for _, rule := range rules {
			if isBaseRule(rule) && rule.Directive == directive {
				group = append(group, rule)
			}
		}
		if len(group) == 0 {
			continue
		}
		wroteAny = true
		fmt.Fprintf(body, "\n#### %s\n", directiveHeading(directive))
		for _, rule := range group {
			writeStandingOrder(body, rule, manifest, titles)
		}
	}
	if !wroteAny {
		body.WriteString("\nNo base standing orders were retained.\n")
	}
}

func writeStandingOrder(
	body *strings.Builder,
	rule rulepack.Rule,
	manifest map[string]rulepack.AcceptedArtifact,
	titles map[string]string,
) {
	fmt.Fprintf(body, "\n##### %s (%s)\n\n", markdownText(rule.Title), inlineCode(rule.ID))
	writeQuotedMarkdown(body, rule.Body)
	if !strings.HasSuffix(rule.Body, "\n") {
		body.WriteString("\n")
	}
	body.WriteString("\n")
	fmt.Fprintf(body, "- Applies to: %s\n", codeList(rule.Scopes))
	writeRelationships(body, manifest[rule.ID], manifest, titles, "")
	fmt.Fprintf(body, "- Category: %s\n", inlineCode(rule.Category))
	fmt.Fprintf(body, "- Canonical rule: %s\n", markdownLink(rule.SourcePath, rule.SourcePath))
	fmt.Fprintf(body, "- Evidence: %s\n", evidenceSourceList(rule.Evidence, false))
}

func writeContextualRules(
	body *strings.Builder,
	rules []rulepack.Rule,
	manifest map[string]rulepack.AcceptedArtifact,
	titles map[string]string,
) {
	contextual := make([]rulepack.Rule, 0)
	for _, rule := range rules {
		if !isBaseRule(rule) {
			contextual = append(contextual, rule)
		}
	}
	if len(contextual) == 0 {
		return
	}
	body.WriteString("\n### Contextual semantic rules\n")
	for _, rule := range contextual {
		fmt.Fprintf(body, "\n#### %s (%s) — %s\n\n", markdownLink(rule.Title, rule.SourcePath), inlineCode(rule.ID), inlineCode(rule.Directive))
		fmt.Fprintf(body, "- Load when: %s\n", lensCodeList(rule.Lenses))
		fmt.Fprintf(body, "- Applies to: %s\n", codeList(rule.Scopes))
		writeRelationships(body, manifest[rule.ID], manifest, titles, "")
		fmt.Fprintf(body, "- Category: %s\n", inlineCode(rule.Category))
		fmt.Fprintf(body, "- Canonical rule: %s\n", markdownLink(rule.SourcePath, rule.SourcePath))
		fmt.Fprintf(body, "- Evidence: %s\n", evidenceSourceList(rule.Evidence, false))
	}
}

func writeVerificationRecipes(
	body *strings.Builder,
	recipes []rulepack.VerificationRecipe,
	manifest map[string]rulepack.AcceptedArtifact,
	titles map[string]string,
) {
	if len(recipes) == 0 {
		return
	}
	body.WriteString("\n### Verification commands\n")
	for _, recipe := range recipes {
		fmt.Fprintf(body, "\n#### %s (%s)\n\n", markdownLink(recipe.Title, recipe.SourcePath), inlineCode(recipe.ID))
		fmt.Fprintf(body, "- When: %s\n", markdownText(strings.TrimSpace(recipe.When)))
		fmt.Fprintf(body, "- Route when: %s\n", lensCodeList(recipe.Lenses))
		fmt.Fprintf(body, "- Applies to: %s\n", codeList(recipe.Scopes))
		writeRelationships(body, manifest[recipe.ID], manifest, titles, "")
		fmt.Fprintf(body, "- Category: %s\n", inlineCode(recipe.Category))
		fmt.Fprintf(body, "- Canonical recipe: %s\n", markdownLink(recipe.SourcePath, recipe.SourcePath))
		fmt.Fprintf(body, "- Evidence: %s\n", evidenceSourceList(recipe.Evidence, false))
		for index, step := range recipe.Steps {
			fmt.Fprintf(body, "\n##### Step %d\n\n", index+1)
			if step.WorkingDirectory != "." {
				fmt.Fprintf(body, "Working directory: %s\n\n", inlineCode(step.WorkingDirectory))
			}
			writeCommandFence(body, step.Run)
			fmt.Fprintf(body, "\nExpected result: %s\n", markdownText(step.ExpectedResult))
		}
	}
}

func writeSkills(
	body *strings.Builder,
	skills []rulepack.Skill,
	manifest map[string]rulepack.AcceptedArtifact,
	titles map[string]string,
) {
	if len(skills) == 0 {
		return
	}
	body.WriteString("\n### Agent Skills\n")
	for _, skill := range skills {
		metadata, exists := manifest[skill.ID]
		if !exists {
			continue
		}
		fmt.Fprintf(body, "\n#### %s (%s)\n\n", markdownLink(titleFromID(skill.ID), skill.SourcePath), inlineCode(skill.ID))
		fmt.Fprintf(body, "%s\n\n", markdownText(strings.TrimSpace(skill.Description)))
		fmt.Fprintf(body, "- Use when: %s\n", lensCodeList(metadata.Lenses))
		fmt.Fprintf(body, "- Applies to: %s\n", codeList(metadata.Scopes))
		writeRelationships(body, metadata, manifest, titles, "")
		fmt.Fprintf(body, "- Category: %s\n", inlineCode(skill.Category))
		fmt.Fprintf(body, "- Evidence: %s\n", evidenceSourceList(metadata.Evidence, false))
	}
}

func writeRelationships(
	body *strings.Builder,
	artifact rulepack.AcceptedArtifact,
	manifest map[string]rulepack.AcceptedArtifact,
	titles map[string]string,
	indent string,
) {
	for _, relatedID := range artifact.RelatedArtifactIDs {
		writeRelatedArtifact(body, relatedID, manifest, titles, indent)
	}
}

func writeRelatedArtifact(
	body *strings.Builder,
	relatedID string,
	manifest map[string]rulepack.AcceptedArtifact,
	titles map[string]string,
	indent string,
) {
	related, exists := manifest[relatedID]
	if !exists {
		return
	}
	label := ""
	switch related.Kind {
	case "rule":
		label = "rule"
	case "verification":
		label = "recipe"
	case "skill":
		label = "skill"
	default:
		return
	}
	title := titles[related.ID]
	if title == "" {
		title = titleFromID(related.ID)
	}
	fmt.Fprintf(body, "%s- Related %s: %s\n", indent, label, markdownLink(title, related.Path))
}

func titleFromID(id string) string {
	title := strings.ReplaceAll(id, "-", " ")
	if title != "" {
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	return title
}

func evidenceSourceList(evidence []rulepack.Evidence, includeRole bool) string {
	values := make([]string, 0, len(evidence))
	seen := make(map[string]struct{}, len(evidence))
	for _, item := range evidence {
		value := item.Path + ":" + item.Lines
		if includeRole {
			value += " (" + item.Role + ")"
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, inlineCode(value))
	}
	return strings.Join(values, ", ")
}

func markdownText(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		"&", "&amp;",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"<", "\\<",
		">", "\\>",
		"#", "\\#",
		"!", "\\!",
		"|", "\\|",
		"~", "\\~",
	)
	return visibleControlCharacters(replacer.Replace(value))
}

func markdownLink(label, relative string) string {
	segments := strings.Split(relative, "/")
	repositoryRelativePrefix := ""
	if segments[0] == "" || strings.Contains(segments[0], ":") {
		repositoryRelativePrefix = "./"
	}
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return "[" + markdownText(label) + "](" + repositoryRelativePrefix + strings.Join(segments, "/") + ")"
}

func inlineCode(value string) string {
	value = visibleControlCharacters(value)
	delimiter := strings.Repeat("`", longestRun(value, '`')+1)
	if strings.HasPrefix(value, "`") || strings.HasSuffix(value, "`") ||
		strings.HasPrefix(value, " ") || strings.HasSuffix(value, " ") {
		return delimiter + " " + value + " " + delimiter
	}
	return delimiter + value + delimiter
}

func visibleControlCharacters(value string) string {
	var result strings.Builder
	for _, character := range value {
		switch character {
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		case '\t':
			result.WriteString(`\t`)
		default:
			if unicode.IsControl(character) || unicode.In(character, unicode.Cf, unicode.Zl, unicode.Zp) {
				fmt.Fprintf(&result, `\u{%04X}`, character)
				continue
			}
			result.WriteRune(character)
		}
	}
	return result.String()
}

func writeQuotedMarkdown(body *strings.Builder, value string) {
	if value == "" {
		return
	}
	body.WriteString("> ")
	for index := 0; index < len(value); index++ {
		body.WriteByte(value[index])
		if value[index] == '\n' && index+1 < len(value) {
			body.WriteString("> ")
		} else if value[index] == '\r' && index+1 < len(value) && value[index+1] != '\n' {
			body.WriteString("> ")
		}
	}
}

func writeCommandFence(body *strings.Builder, command string) {
	fenceLength := longestRun(command, '`') + 1
	if fenceLength < 3 {
		fenceLength = 3
	}
	fence := strings.Repeat("`", fenceLength)
	body.WriteString(fence)
	body.WriteString("\n")
	body.WriteString(command)
	if !strings.HasSuffix(command, "\n") {
		body.WriteString("\n")
	}
	body.WriteString(fence)
	body.WriteString("\n")
}

func longestRun(value string, target byte) int {
	longest := 0
	current := 0
	for index := 0; index < len(value); index++ {
		if value[index] == target {
			current++
			if current > longest {
				longest = current
			}
			continue
		}
		current = 0
	}
	return longest
}

func isBaseRule(rule rulepack.Rule) bool {
	return len(rule.Lenses) == 1 && rule.Lenses[0].Kind == "base"
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

func replaceManagedSection(existing, section []byte) ([]byte, error) {
	start, end, found, err := managedSectionRange(existing)
	if err != nil {
		return nil, err
	}
	if found {
		next := make([]byte, 0, len(existing)-(end-start)+len(section))
		next = append(next, existing[:start]...)
		next = append(next, section...)
		next = append(next, existing[end:]...)
		return next, nil
	}

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

func removeManagedSection(existing []byte) ([]byte, error) {
	start, end, found, err := managedSectionRange(existing)
	if err != nil {
		return nil, err
	}
	if !found {
		return append([]byte(nil), existing...), nil
	}
	removalEnd := end
	if removalEnd+1 == len(existing) && existing[removalEnd] == '\n' {
		removalEnd++
	}
	next := make([]byte, 0, len(existing)-(removalEnd-start))
	next = append(next, existing[:start]...)
	next = append(next, existing[removalEnd:]...)
	return next, nil
}

func managedSectionRange(existing []byte) (int, int, bool, error) {
	startCount := bytes.Count(existing, []byte(StartMarker))
	endCount := bytes.Count(existing, []byte(EndMarker))
	if startCount == 0 && endCount == 0 {
		return 0, 0, false, nil
	}
	if startCount != 1 || endCount != 1 {
		return 0, 0, false, fmt.Errorf("%w: expected exactly one start and one end marker", ErrMarkers)
	}
	start := bytes.Index(existing, []byte(StartMarker))
	end := bytes.Index(existing, []byte(EndMarker))
	if end < start+len(StartMarker) {
		return 0, 0, false, fmt.Errorf("%w: end marker appears before start marker", ErrMarkers)
	}
	end += len(EndMarker)
	if err := verifyExistingSection(existing[start:end]); err != nil {
		return 0, 0, false, err
	}
	return start, end, true, nil
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
		return fmt.Errorf("%w: edit canonical artifact sources and report.md instead of editing the generated section", ErrDrift)
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
		quoted = append(quoted, inlineCode(value))
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
