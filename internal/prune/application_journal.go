package prune

import (
	"errors"
	"fmt"
	"os"
)

const journalSchema = "ssb.dev/prune-application-journal/v1"

type journalEntry struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"`
	Mode    uint32 `json:"mode,omitempty"`
	Content []byte `json:"content,omitempty"`
}

type applicationJournal struct {
	Schema             string         `json:"schema"`
	ReviewID           string         `json:"review_id"`
	ProposalDigest     string         `json:"proposal_digest"`
	PlanDigest         string         `json:"plan_digest"`
	Entries            []journalEntry `json:"entries"`
	CreatedDirectories []string       `json:"created_directories,omitempty"`
}

func cleanupApplicationJournal(store *reviewStore, reviewID string) error {
	if err := removeApplicationJournal(store); err != nil {
		return fmt.Errorf(
			"application succeeded but recovery journal cleanup failed: %w; run ssb prune recover --review %s",
			err,
			reviewID,
		)
	}
	return nil
}

func captureJournal(
	repoRoot string,
	review Review,
	plan applicationPlan,
	operations []operation,
) (applicationJournal, error) {
	journal := applicationJournal{
		Schema:         journalSchema,
		ReviewID:       review.Context.ReviewID,
		ProposalDigest: review.ProposalDigest,
		PlanDigest:     plan.PlanDigest,
		Entries:        make([]journalEntry, 0, len(operations)),
	}
	createdDirectories, err := missingApplicationDirectories(repoRoot, operations)
	if err != nil {
		return applicationJournal{}, err
	}
	journal.CreatedDirectories = createdDirectories
	for _, operation := range operations {
		if err := safeApplicationTarget(repoRoot, operation.Path); err != nil {
			return applicationJournal{}, err
		}
		target, err := resolvePortablePath(repoRoot, operation.Path)
		if err != nil {
			return applicationJournal{}, err
		}
		if operation.ExpectedAbsent {
			if _, err := os.Lstat(target); err == nil {
				return applicationJournal{}, fmt.Errorf("new target %s appeared after preflight", operation.Path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return applicationJournal{}, fmt.Errorf("inspect expected-absent target %s: %w", operation.Path, err)
			}
			journal.Entries = append(journal.Entries, journalEntry{Path: operation.Path})
			continue
		}
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			return applicationJournal{}, fmt.Errorf(
				"application target %s disappeared after preflight",
				operation.Path,
			)
		}
		if err != nil {
			return applicationJournal{}, fmt.Errorf("inspect application target %s: %w", operation.Path, err)
		}
		if !info.Mode().IsRegular() {
			return applicationJournal{}, fmt.Errorf("application target %s is not a regular file", operation.Path)
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return applicationJournal{}, fmt.Errorf("capture application target %s: %w", operation.Path, err)
		}
		if digestBytes(content) != operation.Prestate.SHA256 ||
			materializedMode(info.Mode()) != operation.Prestate.Mode {
			return applicationJournal{}, fmt.Errorf(
				"application target %s changed after preflight",
				operation.Path,
			)
		}
		journal.Entries = append(journal.Entries, journalEntry{
			Path:    operation.Path,
			Existed: true,
			Mode:    uint32(info.Mode().Perm()),
			Content: content,
		})
	}
	return journal, nil
}

func (publication applicationPublicationOps) restoreJournal(repoRoot string, journal applicationJournal, poststates map[string]string) error {
	paths := make(map[string]struct{}, len(journal.Entries))
	for _, entry := range journal.Entries {
		paths[entry.Path] = struct{}{}
	}
	return publication.restoreJournalPaths(
		repoRoot,
		journal,
		paths,
		poststates,
	)
}

func (publication applicationPublicationOps) restoreJournalPaths(
	repoRoot string,
	journal applicationJournal,
	paths map[string]struct{},
	poststates map[string]string,
) error {
	for _, entry := range journal.Entries {
		if _, restore := paths[entry.Path]; !restore {
			continue
		}
		if err := safeApplicationTarget(repoRoot, entry.Path); err != nil {
			return err
		}
		target, err := resolvePortablePath(repoRoot, entry.Path)
		if err != nil {
			return err
		}
		expectedPoststate, hasPoststate := poststates[entry.Path]
		if !hasPoststate {
			return fmt.Errorf("missing approved poststate for %s", entry.Path)
		}
		if err := publication.restoreJournalEntry(target, entry, expectedPoststate); err != nil {
			return err
		}
	}
	return removeApplicationDirectories(repoRoot, journal.CreatedDirectories)
}

func operationPoststates(operations []operation) map[string]string {
	result := make(map[string]string, len(operations))
	for _, operation := range operations {
		if operation.Kind == "write" {
			result[operation.Path] = operation.Poststate.SHA256
		} else {
			result[operation.Path] = ""
		}
	}
	return result
}

func removeApplicationJournal(store *reviewStore) error {
	return store.Remove("application-journal.json")
}
