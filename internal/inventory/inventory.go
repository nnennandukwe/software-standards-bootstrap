// Package inventory builds a bounded, deterministic index of safe text blobs
// from a commit. It does not infer standards or execute repository code.
package inventory

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const Version = "ssb-inventory-v1"

// Limits bound repository resource use during inspection.
type Limits struct {
	MaxFiles      int   `json:"max_files"`
	MaxTotalBytes int64 `json:"max_total_bytes"`
	MaxFileBytes  int64 `json:"max_file_bytes"`
}

// DefaultLimits are deliberately conservative for an agent-oriented evidence
// inventory. Any report cut short by MaxFiles or MaxTotalBytes says so.
func DefaultLimits() Limits {
	return Limits{
		MaxFiles:      20_000,
		MaxTotalBytes: 25 << 20,
		MaxFileBytes:  1 << 20,
	}
}

// Exclusions counts tracked entries that were deliberately not read.
type Exclusions struct {
	Binary     int `json:"binary"`
	Generated  int `json:"generated"`
	Oversized  int `json:"oversized"`
	SecretLike int `json:"secret_like"`
	Symlink    int `json:"symlink"`
	Submodule  int `json:"submodule"`
	VendorTree int `json:"vendor_or_generated_tree"`
	NonRegular int `json:"non_regular"`
}

// File is a safe, tracked text blob available for targeted semantic reads.
type File struct {
	Path     string `json:"path"`
	BlobOID  string `json:"blob_oid"`
	Bytes    int64  `json:"bytes"`
	Lines    int    `json:"lines"`
	Language string `json:"language,omitempty"`
	SHA256   string `json:"sha256"`
}

// Report is stable for a given commit and set of limits.
type Report struct {
	SchemaVersion    int        `json:"schema_version"`
	InventoryVersion string     `json:"inventory_version"`
	BaselineCommit   string     `json:"baseline_commit"`
	Limits           Limits     `json:"limits"`
	Files            []File     `json:"files"`
	Excluded         Exclusions `json:"excluded"`
	IndexedBytes     int64      `json:"indexed_bytes"`
	Truncated        bool       `json:"truncated"`
	TruncationReason string     `json:"truncation_reason,omitempty"`
}

type treeEntry struct {
	mode string
	kind string
	oid  string
	size int64
	path string
}

// ReadEvidence returns one baseline file only when it is eligible for the
// inspection inventory. Rule validation uses this same boundary so a rule
// cannot cite a secret-like, generated, binary, vendored, or oversized file
// that the host agent was never allowed to inspect.
func ReadEvidence(ctx context.Context, repo *workspace.Repository, filePath string) ([]byte, error) {
	if excludedTree(filePath) {
		return nil, fmt.Errorf("evidence path %q is excluded as a vendor/generated tree", filePath)
	}
	if secretLike(filePath) {
		return nil, fmt.Errorf("evidence path %q is excluded as secret-like", filePath)
	}
	entry, exists, err := repo.EntryAtBaseline(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("evidence path %q does not exist at baseline %s", filePath, repo.Baseline())
	}
	if (entry.Mode != "100644" && entry.Mode != "100755") || entry.Kind != "blob" {
		return nil, fmt.Errorf("evidence path %q is not a tracked regular file", filePath)
	}
	maxBytes := DefaultLimits().MaxFileBytes
	if entry.Size < 0 || entry.Size > maxBytes {
		return nil, fmt.Errorf("evidence path %q is larger than %d bytes", filePath, maxBytes)
	}
	content, err := repo.ReadBaselineFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	if binary(content) {
		return nil, fmt.Errorf("evidence path %q is excluded as binary content", filePath)
	}
	if generated(content) {
		return nil, fmt.Errorf("evidence path %q is excluded as generated content", filePath)
	}
	return content, nil
}

// Scan reads blobs from the pinned baseline commit, not from the worktree.
func Scan(ctx context.Context, repo *workspace.Repository, limits Limits) (Report, error) {
	limits = normalizedLimits(limits)
	report := Report{
		SchemaVersion:    1,
		InventoryVersion: Version,
		BaselineCommit:   repo.Baseline(),
		Limits:           limits,
		Files:            make([]File, 0),
	}

	rawTree, err := repo.Git(ctx, "ls-tree", "-rz", "-l", "--full-tree", repo.Baseline())
	if err != nil {
		return Report{}, fmt.Errorf("list baseline tree: %w", err)
	}
	entries, err := parseTree(rawTree)
	if err != nil {
		return Report{}, err
	}

	candidates := make([]treeEntry, 0, len(entries))
	for _, entry := range entries {
		switch entry.mode {
		case "120000":
			report.Excluded.Symlink++
			continue
		case "160000":
			report.Excluded.Submodule++
			continue
		case "100644", "100755":
			if entry.kind != "blob" {
				report.Excluded.NonRegular++
				continue
			}
		default:
			report.Excluded.NonRegular++
			continue
		}

		if excludedTree(entry.path) {
			report.Excluded.VendorTree++
			continue
		}
		if secretLike(entry.path) {
			report.Excluded.SecretLike++
			continue
		}
		if entry.size < 0 || entry.size > limits.MaxFileBytes {
			report.Excluded.Oversized++
			continue
		}
		candidates = append(candidates, entry)
	}

	scanComplete := true
	for start := 0; start < len(candidates); {
		end := batchEnd(candidates, start)
		contents, err := readBlobBatch(ctx, repo, candidates[start:end])
		if err != nil {
			return Report{}, err
		}
		for index, content := range contents {
			entry := candidates[start+index]
			if binary(content) {
				report.Excluded.Binary++
				continue
			}
			if generated(content) {
				report.Excluded.Generated++
				continue
			}
			if len(report.Files) >= limits.MaxFiles {
				report.Truncated = true
				report.TruncationReason = fmt.Sprintf("inventory stopped at max_files=%d", limits.MaxFiles)
				scanComplete = false
				break
			}
			if report.IndexedBytes+entry.size > limits.MaxTotalBytes {
				report.Truncated = true
				report.TruncationReason = fmt.Sprintf("inventory stopped at max_total_bytes=%d", limits.MaxTotalBytes)
				scanComplete = false
				break
			}

			sum := sha256.Sum256(content)
			report.Files = append(report.Files, File{
				Path:     entry.path,
				BlobOID:  entry.oid,
				Bytes:    entry.size,
				Lines:    lineCount(content),
				Language: language(entry.path),
				SHA256:   "sha256:" + hex.EncodeToString(sum[:]),
			})
			report.IndexedBytes += entry.size
		}
		if !scanComplete {
			break
		}
		start = end
	}
	if err := repo.VerifyInspectSnapshot(ctx); err != nil {
		return Report{}, err
	}
	return report, nil
}

func batchEnd(entries []treeEntry, start int) int {
	const (
		maxEntries = 128
		maxBytes   = int64(8 << 20)
	)
	end := start
	var size int64
	for end < len(entries) && end-start < maxEntries {
		next := entries[end].size
		if end > start && size+next > maxBytes {
			break
		}
		size += next
		end++
	}
	return end
}

func readBlobBatch(ctx context.Context, repo *workspace.Repository, entries []treeEntry) ([][]byte, error) {
	var input strings.Builder
	for _, entry := range entries {
		input.WriteString(entry.oid)
		input.WriteByte('\n')
	}
	raw, err := repo.GitWithInput(ctx, []byte(input.String()), "cat-file", "--batch")
	if err != nil {
		return nil, fmt.Errorf("read baseline blob batch: %w", err)
	}
	reader := bufio.NewReader(bytes.NewReader(raw))
	contents := make([][]byte, 0, len(entries))
	for _, entry := range entries {
		header, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read baseline blob header for %q: %w", entry.path, err)
		}
		fields := strings.Fields(strings.TrimSuffix(header, "\n"))
		if len(fields) != 3 || fields[0] != entry.oid || fields[1] != "blob" {
			return nil, fmt.Errorf("unexpected baseline blob header for %q: %q", entry.path, strings.TrimSpace(header))
		}
		size, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || size != entry.size {
			return nil, fmt.Errorf("baseline blob %q changed size while reading", entry.path)
		}
		content := make([]byte, size)
		if _, err := io.ReadFull(reader, content); err != nil {
			return nil, fmt.Errorf("read baseline blob %q: %w", entry.path, err)
		}
		delimiter, err := reader.ReadByte()
		if err != nil || delimiter != '\n' {
			return nil, fmt.Errorf("read baseline blob delimiter for %q", entry.path)
		}
		contents = append(contents, content)
	}
	if trailing, err := reader.ReadByte(); err == nil {
		return nil, fmt.Errorf("unexpected trailing byte %q from git cat-file --batch", trailing)
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read git cat-file trailer: %w", err)
	}
	return contents, nil
}

func normalizedLimits(limits Limits) Limits {
	defaults := DefaultLimits()
	if limits.MaxFiles <= 0 {
		limits.MaxFiles = defaults.MaxFiles
	}
	if limits.MaxTotalBytes <= 0 {
		limits.MaxTotalBytes = defaults.MaxTotalBytes
	}
	if limits.MaxFileBytes <= 0 {
		limits.MaxFileBytes = defaults.MaxFileBytes
	}
	return limits
}

func parseTree(raw []byte) ([]treeEntry, error) {
	records := bytes.Split(raw, []byte{0})
	entries := make([]treeEntry, 0, len(records))
	for _, record := range records {
		if len(record) == 0 {
			continue
		}
		tab := bytes.IndexByte(record, '\t')
		if tab < 0 {
			return nil, fmt.Errorf("parse baseline tree: missing path separator")
		}
		fields := strings.Fields(string(record[:tab]))
		if len(fields) != 4 {
			return nil, fmt.Errorf("parse baseline tree entry for %q", record[tab+1:])
		}
		size := int64(-1)
		if fields[3] != "-" {
			parsed, err := strconv.ParseInt(fields[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse size for %q: %w", record[tab+1:], err)
			}
			size = parsed
		}
		entryPath := string(record[tab+1:])
		if path.IsAbs(entryPath) || path.Clean(entryPath) != entryPath || entryPath == "." ||
			entryPath == ".." || strings.HasPrefix(entryPath, "../") {
			return nil, fmt.Errorf("unsafe path %q in baseline tree", entryPath)
		}
		entries = append(entries, treeEntry{
			mode: fields[0],
			kind: fields[1],
			oid:  fields[2],
			size: size,
			path: entryPath,
		})
	}
	return entries, nil
}

func excludedTree(filePath string) bool {
	for _, component := range strings.Split(strings.ToLower(filePath), "/") {
		switch component {
		case "vendor", "node_modules", ".venv", "venv", "dist", "build",
			"target", "coverage", "third_party", "third-party", "external",
			"generated", "gen":
			return true
		}
	}
	return false
}

func secretLike(filePath string) bool {
	base := strings.ToLower(path.Base(filePath))
	if base == ".env.example" || base == ".env.sample" || base == ".env.template" {
		return false
	}
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	switch base {
	case "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "credentials",
		"credentials.json", "secrets.yml", "secrets.yaml", ".npmrc", ".pypirc":
		return true
	}
	switch strings.ToLower(path.Ext(base)) {
	case ".pem", ".key", ".p12", ".pfx", ".jks", ".keystore":
		return true
	}
	return strings.Contains(base, "secret") || strings.Contains(base, "credential")
}

func binary(content []byte) bool {
	sample := content
	if len(sample) > 8<<10 {
		sample = sample[:8<<10]
	}
	return bytes.IndexByte(sample, 0) >= 0
}

func generated(content []byte) bool {
	sample := content
	if len(sample) > 16<<10 {
		sample = sample[:16<<10]
	}
	lower := bytes.ToLower(sample)
	return bytes.Contains(lower, []byte("@generated")) ||
		bytes.Contains(lower, []byte("code generated")) && bytes.Contains(lower, []byte("do not edit"))
}

func lineCount(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	lines := bytes.Count(content, []byte{'\n'})
	if content[len(content)-1] != '\n' {
		lines++
	}
	return lines
}

func language(filePath string) string {
	switch strings.ToLower(path.Ext(filePath)) {
	case ".go":
		return "Go"
	case ".py":
		return "Python"
	case ".js", ".mjs", ".cjs":
		return "JavaScript"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".kt", ".kts":
		return "Kotlin"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".cs":
		return "C#"
	case ".c", ".h":
		return "C"
	case ".cc", ".cpp", ".cxx", ".hpp":
		return "C++"
	case ".swift":
		return "Swift"
	case ".sh", ".bash", ".zsh":
		return "Shell"
	case ".md", ".mdx":
		return "Markdown"
	case ".yaml", ".yml":
		return "YAML"
	case ".json":
		return "JSON"
	case ".toml":
		return "TOML"
	default:
		return ""
	}
}
