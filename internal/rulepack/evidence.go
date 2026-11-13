package rulepack

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func validateDerivationEvidence(
	ctx context.Context,
	repo *workspace.Repository,
	sourcePath string,
	derivation string,
	evidence []Evidence,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field,
			message,
			recovery,
		))
	}
	if derivation != "extracted" && derivation != "inferred" {
		add("derivation", "derivation must be extracted or inferred", "record how the artifact was derived from repository evidence")
	}
	if len(evidence) == 0 {
		add("evidence", "at least one exact evidence citation is required", "cite eligible baseline lines and their excerpt hash")
	}
	seenLocations := make(map[string]struct{}, len(evidence))
	demonstrates := 0
	demonstratedFiles := make(map[string]struct{})
	hasAuthority := false
	for index, item := range evidence {
		field := fmt.Sprintf("evidence[%d]", index)
		switch item.Role {
		case "declares", "enforces":
			hasAuthority = true
		case "demonstrates":
			demonstrates++
			demonstratedFiles[item.Path] = struct{}{}
		default:
			add(field+".role", fmt.Sprintf("evidence role %q is not supported", item.Role), "use declares, demonstrates, or enforces")
		}
		diagnostics = append(diagnostics, validateEvidence(
			ctx,
			repo,
			sourcePath,
			field,
			item,
		)...)
		location := item.Path + "\x00" + item.Lines
		if _, duplicate := seenLocations[location]; duplicate {
			add("evidence", fmt.Sprintf("duplicate evidence location %s:%s", item.Path, item.Lines), "cite each distinct occurrence once")
		}
		seenLocations[location] = struct{}{}
	}
	if derivation == "extracted" && !hasAuthority {
		add("evidence", "extracted artifacts require at least one declares or enforces citation", "cite the repository-maintained obligation or active enforcement mechanism")
	}
	if derivation == "inferred" && (demonstrates < 3 || len(demonstratedFiles) < 2) {
		add("evidence", "inferred artifacts require three demonstrates citations across at least two files", "add consistent implementation occurrences or remove the candidate")
	}
	return diagnostics
}

func validateEvidence(ctx context.Context, repo *workspace.Repository, sourcePath, field string, evidence Evidence) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field,
			message,
			recovery,
		))
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
		add(fmt.Sprintf(
			"excerpt hash does not match %s lines %s; expected %s",
			evidence.Path,
			evidence.Lines,
			actual,
		), "refresh the evidence after inspecting the exact baseline bytes")
	}
	return diagnostics
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
		return nil, fmt.Errorf(
			"line range %d-%d exceeds file length %d",
			start,
			end,
			len(lines),
		)
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
