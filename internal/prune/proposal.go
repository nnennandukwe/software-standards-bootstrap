package prune

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// ValidateProposal validates semantic output without treating it as approval.
func ValidateProposal(context Context, proposal Proposal, reviewRoot string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(actionID, field, message, recovery string) {
		itemPath := "proposal.yaml"
		if actionID != "" {
			itemPath += "#" + actionID
		}
		diagnostics = append(diagnostics, Diagnostic{
			Path:     itemPath,
			Field:    field,
			Message:  message,
			Recovery: recovery,
		})
	}
	if proposal.Schema != ProposalSchema {
		add("", "schema", "proposal schema must be "+ProposalSchema, "update the proposal schema")
	}
	if proposal.ReviewID != context.ReviewID {
		add("", "review_id", "proposal review_id does not match the context", "regenerate the proposal for this review")
	}
	if proposal.ContextDigest != context.ContextDigest {
		add("", "context_digest", "proposal context digest is stale", "rerun ssb prune inspect and regenerate the proposal")
	}
	artifacts := make(map[string]Artifact)
	for _, artifact := range context.Artifacts {
		artifacts[artifactKey(artifact.Kind, artifact.Path)] = artifact
	}
	capabilities := make(map[string]Capability)
	for _, capability := range context.Capabilities.Capabilities {
		capabilities[capability.ID] = capability
	}
	inventoryFiles := make(map[string]struct {
		SHA256 string
		Lines  int
	})
	for _, file := range context.Inventory.Files {
		inventoryFiles[file.Path] = struct {
			SHA256 string
			Lines  int
		}{SHA256: file.SHA256, Lines: file.Lines}
	}
	covered := make(map[string]string)
	actionIDs := make(map[string]struct{})
	targets := make(map[string]string)
	checks := make(map[string]string)
	candidateBudgetOK := validateCandidateBudget(context, proposal, reviewRoot, add)
	for index, action := range proposal.Actions {
		actionID := action.ID
		if actionID == "" {
			actionID = fmt.Sprintf("actions[%d]", index)
		}
		if !stableIDPattern.MatchString(action.ID) {
			add(actionID, "id", "action id must be lower-case kebab-case", "choose a stable action id")
		}
		if _, duplicate := actionIDs[action.ID]; duplicate {
			add(actionID, "id", "duplicate action id "+action.ID, "give every action a unique id")
		}
		actionIDs[action.ID] = struct{}{}
		if strings.TrimSpace(action.Rationale) == "" {
			add(actionID, "rationale", "every disposition requires rationale", "explain why the evidence supports this disposition")
		}
		switch action.Confidence {
		case ConfidenceLow, ConfidenceMedium, ConfidenceHigh:
		default:
			add(actionID, "confidence", "confidence must be low, medium, or high", "record an honest confidence band")
		}

		validateActionShape(action, reviewRoot, context.Inventory.Limits.MaxFileBytes, candidateBudgetOK, runtime.GOOS, add)
		if action.Target != nil {
			for _, targetPath := range candidateTargetPaths(*action.Target) {
				if prior, duplicate := targets[targetPath]; duplicate {
					add(actionID, "target.target_path", "target path is also written by action "+prior, "choose one owning action for each target")
				} else {
					targets[targetPath] = action.ID
				}
			}
			for _, artifact := range context.Artifacts {
				if artifact.Path != action.Target.TargetPath {
					continue
				}
				owned := false
				for _, source := range action.Sources {
					owned = owned || source.Path == artifact.Path
				}
				if !owned {
					add(actionID, "target.target_path", "target would overwrite artifact "+artifact.ID+" owned by another action", "include the existing target as a source of this action")
				}
			}
		}
		for _, source := range action.Sources {
			key := artifactKey(source.Kind, source.Path)
			artifact, exists := artifacts[key]
			if !exists || source.ID != artifact.ID || source.SHA256 != artifact.SHA256 {
				add(actionID, "sources", "source does not match an exact context artifact: "+source.Path, "rebuild the proposal from the current context")
				continue
			}
			if prior, duplicate := covered[key]; duplicate {
				add(actionID, "sources", "artifact "+source.ID+" also appears in action "+prior, "cover every artifact exactly once")
			} else {
				covered[key] = action.ID
			}
			if artifact.Origin == OriginUnknown && action.Disposition != DispositionUnableToDetermine {
				add(actionID, "disposition", "artifact "+artifact.ID+" has unknown provenance and must remain unable-to-determine", "add explicit provenance evidence or change the disposition")
			}
		}
		if action.Disposition == DispositionUnableToDetermine {
			if len(action.UnresolvedQuestions) == 0 {
				add(actionID, "unresolved_questions", "unable-to-determine requires at least one unresolved question", "name the missing inventory, provenance, capability, or repository evidence")
			}
			continue
		}
		if len(action.RepositoryEvidence) == 0 {
			add(actionID, "repository_evidence", "every actionable disposition requires repository evidence", "cite exact current repository evidence")
		}
		for _, evidence := range action.RepositoryEvidence {
			if !safeRelativePath(evidence.Path) || strings.TrimSpace(evidence.Lines) == "" || !validDigest(evidence.SHA256) {
				add(actionID, "repository_evidence", "repository evidence must contain a safe path, line range, and digest", "cite exact current repository evidence")
				continue
			}
			file, exists := inventoryFiles[evidence.Path]
			if !exists {
				add(actionID, "repository_evidence", "repository evidence "+evidence.Path+" is absent from the complete inventory", "cite an eligible file in context.json")
				continue
			}
			_, end, err := proposalLineRange(evidence.Lines)
			if err != nil || end > file.Lines {
				add(actionID, "repository_evidence", "repository evidence has an invalid line range for "+evidence.Path, "use a one-based range within the inventoried file")
			}
			if evidence.SHA256 != file.SHA256 {
				add(actionID, "repository_evidence", "repository evidence digest does not match the inventoried file "+evidence.Path, "use the exact sha256 from context.json")
			}
		}
		if len(action.CapabilityRefs) == 0 {
			add(actionID, "capability_refs", "every actionable disposition requires capability evidence", "reference a supported or unsupported observed capability")
		}
		for _, capabilityID := range action.CapabilityRefs {
			capability, exists := capabilities[capabilityID]
			if !exists {
				add(actionID, "capability_refs", "unknown capability "+capabilityID, "reference a capability from the selected profile")
				continue
			}
			if capability.Status == CapabilityUnknown {
				add(actionID, "capability_refs", "capability "+capabilityID+" is unknown", "use unable-to-determine or collect conformance evidence")
			}
		}
		for _, check := range action.RequiredVerification {
			if !stableIDPattern.MatchString(check.ID) || strings.TrimSpace(check.Command) == "" {
				add(actionID, "required_verification", "verification checks require a stable id and exact command", "record the external command whose receipt will be required")
				continue
			}
			if prior, duplicate := checks[check.ID]; duplicate && prior != check.Command {
				add(actionID, "required_verification", "verification check "+check.ID+" has conflicting commands", "use one exact command for each check id")
			}
			checks[check.ID] = check.Command
		}
		if len(action.RequiredVerification) == 0 {
			add(actionID, "required_verification", "every actionable disposition requires an external verification receipt", "map at least one exact check without executing it")
		}
	}
	for key, artifact := range artifacts {
		if _, exists := covered[key]; !exists {
			add("", "actions", "artifact "+artifact.ID+" is not covered by any proposal action", "cover every rule and repository skill exactly once")
		}
	}
	for _, action := range proposal.Actions {
		for _, dependency := range action.Dependencies {
			if _, exists := actionIDs[dependency]; !exists {
				add(action.ID, "dependencies", "unknown action dependency "+dependency, "reference an action in this proposal")
			}
		}
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path == diagnostics[j].Path {
			return diagnostics[i].Field < diagnostics[j].Field
		}
		return diagnostics[i].Path < diagnostics[j].Path
	})
	return diagnostics
}

func validateCandidateBudget(
	context Context,
	proposal Proposal,
	reviewRoot string,
	add func(string, string, string, string),
) bool {
	count := 0
	var total int64
	for _, action := range proposal.Actions {
		if action.Target == nil {
			continue
		}
		sources := []string{action.Target.SourcePath}
		for _, supporting := range action.Target.SupportingFiles {
			sources = append(sources, supporting.SourcePath)
		}
		for _, source := range sources {
			if !safeRelativePath(source) {
				continue
			}
			info, err := os.Lstat(filepath.Join(reviewRoot, filepath.FromSlash(source)))
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			count++
			if info.Size() > context.Inventory.Limits.MaxCandidateBytes-total {
				total = context.Inventory.Limits.MaxCandidateBytes + 1
			} else {
				total += info.Size()
			}
		}
	}
	limits := context.Inventory.Limits
	if count == 0 {
		return true
	}
	if limits.MaxCandidateFiles <= 0 || limits.MaxCandidateBytes <= 0 ||
		count > limits.MaxCandidateFiles || total > limits.MaxCandidateBytes {
		add("", "candidates", "candidate bundle exceeds the review file or byte boundary", "reduce the candidates or create a new review with evidence-backed limits")
		return false
	}
	return true
}

func proposalLineRange(value string) (int, int, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid line range")
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.Atoi(parts[1])
	if err != nil || start < 1 || end < start {
		return 0, 0, fmt.Errorf("invalid line range")
	}
	return start, end, nil
}

func validateActionShape(
	action Action,
	reviewRoot string,
	maxFileBytes int64,
	candidateBudgetOK bool,
	goos string,
	add func(string, string, string, string),
) {
	sourceCount := len(action.Sources)
	switch action.Disposition {
	case DispositionKeep, DispositionRemove, DispositionUnableToDetermine:
		if sourceCount != 1 || action.Target != nil {
			add(action.ID, "disposition", action.Disposition+" requires exactly one source and no target", "correct the action cardinality")
		}
	case DispositionUpdate:
		if sourceCount != 1 || action.Target == nil {
			add(action.ID, "disposition", "update requires exactly one source and one complete target", "add the complete replacement candidate")
		}
	case DispositionConsolidate:
		if sourceCount < 2 || action.Target == nil {
			add(action.ID, "disposition", "consolidate requires at least two sources and one complete target", "add every source and the replacement candidate")
		}
		if sourceCount > 1 {
			kind := action.Sources[0].Kind
			for _, source := range action.Sources[1:] {
				if source.Kind != kind {
					add(action.ID, "sources", "consolidation sources must use the same artifact kind", "keep rule-skill relationships as dependencies")
					break
				}
			}
		}
	default:
		add(action.ID, "disposition", "unsupported disposition "+action.Disposition, "use keep, update, consolidate, remove, or unable-to-determine")
	}
	if action.Target == nil {
		return
	}
	if len(action.Sources) != 0 && action.Target.Kind != action.Sources[0].Kind {
		add(action.ID, "target.kind", "replacement kind must match its source kind", "keep rule and skill lifecycle actions separate")
	}
	if action.Target.Kind != ArtifactRule && action.Target.Kind != ArtifactSkill {
		add(action.ID, "target.kind", "target kind must be rule or skill", "use the canonical artifact kind")
	}
	if err := validateCandidateMode(action.Target.Mode, goos); err != nil {
		add(action.ID, "target.mode", err.Error(), candidateModeRecovery(action.Target.Mode, goos))
	}
	if action.Target.Kind == ArtifactRule && len(action.Target.SupportingFiles) != 0 {
		add(action.ID, "target.supporting_files", "rule candidates cannot contain skill supporting files", "remove supporting_files from the rule target")
	}
	if !validTargetPath(*action.Target) {
		add(action.ID, "target.target_path", "target path is outside the canonical rule or skill location", "choose a canonical repository path")
	}
	if !safeRelativePath(action.Target.SourcePath) ||
		!strings.HasPrefix(action.Target.SourcePath, "candidates/"+action.ID+"/") {
		add(action.ID, "target.source_path", "candidate source must stay inside candidates/"+action.ID, "store a complete replacement inside the action candidate directory")
		return
	}
	if candidateBudgetOK {
		validateCandidateFile(action.ID, "target", action.Target.SourcePath, action.Target.SHA256, reviewRoot, maxFileBytes, add)
	}
	seenSupporting := make(map[string]struct{})
	skillRoot := path.Dir(action.Target.TargetPath) + "/"
	for index, supporting := range action.Target.SupportingFiles {
		field := fmt.Sprintf("target.supporting_files[%d]", index)
		if action.Target.Kind != ArtifactSkill ||
			!safeRelativePath(supporting.TargetPath) ||
			!strings.HasPrefix(supporting.TargetPath, skillRoot) ||
			supporting.TargetPath == action.Target.TargetPath {
			add(action.ID, field+".target_path", "supporting target must be a file beneath the replacement skill directory", "choose a safe path inside "+skillRoot)
		}
		if _, duplicate := seenSupporting[supporting.TargetPath]; duplicate {
			add(action.ID, field+".target_path", "duplicate supporting target "+supporting.TargetPath, "write each replacement file exactly once")
		}
		seenSupporting[supporting.TargetPath] = struct{}{}
		if err := validateCandidateMode(supporting.Mode, goos); err != nil {
			add(action.ID, field+".mode", err.Error(), candidateModeRecovery(supporting.Mode, goos))
		}
		if !safeRelativePath(supporting.SourcePath) ||
			!strings.HasPrefix(supporting.SourcePath, "candidates/"+action.ID+"/") {
			add(action.ID, field+".source_path", "supporting candidate must stay inside candidates/"+action.ID, "store the complete supporting file inside the action candidate directory")
			continue
		}
		if candidateBudgetOK {
			validateCandidateFile(action.ID, field, supporting.SourcePath, supporting.SHA256, reviewRoot, maxFileBytes, add)
		}
	}
}

func validateCandidateMode(mode, goos string) error {
	if mode != "100644" && mode != "100755" {
		return fmt.Errorf("candidate mode must be 100644 or 100755")
	}
	if goos == "windows" && mode == "100755" {
		return fmt.Errorf("cannot materialize Git executable mode 100755 on Windows without changing the index")
	}
	return nil
}

func candidateModeRecovery(mode, goos string) string {
	if goos == "windows" && mode == "100755" {
		return "apply this review from a POSIX host, or use 100644 only when executable intent is not required; ssb will not stage an index-only chmod"
	}
	return "record the intended tracked regular-file mode"
}

func validateCandidateFile(
	actionID, field, sourcePath, expectedDigest, reviewRoot string,
	maxFileBytes int64,
	add func(string, string, string, string),
) {
	candidatePath := filepath.Join(reviewRoot, filepath.FromSlash(sourcePath))
	if err := requireRegularBundleFile(reviewRoot, candidatePath); err != nil {
		add(actionID, field+".source_path", "unsafe candidate: "+err.Error(), "use a regular file inside the review bundle without symlink components")
		return
	}
	info, err := os.Stat(candidatePath)
	if err != nil {
		add(actionID, field+".source_path", "inspect candidate: "+err.Error(), "create the complete replacement candidate")
		return
	}
	if maxFileBytes <= 0 || info.Size() > maxFileBytes {
		add(actionID, field+".source_path", "candidate exceeds the review max_file_bytes boundary", "reduce the candidate or create a new review with an evidence-backed limit")
		return
	}
	content, err := os.ReadFile(candidatePath)
	if err != nil {
		add(actionID, field+".source_path", "read candidate: "+err.Error(), "create the complete replacement candidate")
		return
	}
	if digestBytes(content) != expectedDigest {
		add(actionID, field+".sha256", "candidate digest does not match "+sourcePath, "refresh the proposal from the exact candidate bytes")
	}
}

func candidateTargetPaths(target CandidateRef) []string {
	result := []string{target.TargetPath}
	for _, supporting := range target.SupportingFiles {
		result = append(result, supporting.TargetPath)
	}
	return result
}

func requireRegularBundleFile(root, candidate string) error {
	relative, err := filepath.Rel(root, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes the review bundle")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path contains a symlink")
		}
	}
	info, err := os.Lstat(candidate)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("candidate is not a regular file")
	}
	return nil
}

func validTargetPath(target CandidateRef) bool {
	if !safeRelativePath(target.TargetPath) || !stableIDPattern.MatchString(target.ID) {
		return false
	}
	switch target.Kind {
	case ArtifactRule:
		return target.TargetPath == path.Join(".software-standards/rules", target.ID+".md")
	case ArtifactSkill:
		return target.TargetPath == path.Join(".agents/skills", target.ID, "SKILL.md")
	default:
		return false
	}
}

func artifactKey(kind, artifactPath string) string {
	return kind + "\x00" + artifactPath
}
