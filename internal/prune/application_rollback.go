package prune

import (
	"errors"
	"fmt"
	"os"
)

func (publication applicationPublicationOps) restoreJournalEntry(target string, entry journalEntry, expectedPoststate string) error {
	if err := publication.reconcilePublicationClaim(target, entry, expectedPoststate); err != nil {
		return err
	}
	preClaim := claimPathForTarget(target, prestateClaim)
	postClaim := claimPathForTarget(target, poststateClaim)
	preExists, err := regularFileExists(preClaim)
	if err != nil {
		return err
	}
	postExists, err := regularFileExists(postClaim)
	if err != nil {
		return err
	}
	if preExists && postExists {
		if !entry.Existed {
			return fmt.Errorf("unexpected prestate claim for new target %s", entry.Path)
		}
		if digest, digestErr := fileDigest(preClaim); digestErr != nil || digest != digestBytes(entry.Content) {
			return fmt.Errorf("prestate claim for %s is invalid", entry.Path)
		}
		if digest, digestErr := fileDigest(postClaim); digestErr != nil || digest != expectedPoststate {
			return fmt.Errorf("poststate claim for %s is invalid", entry.Path)
		}
		targetDigest, absent, digestErr := currentFileDigest(target)
		if digestErr != nil {
			return digestErr
		}
		switch {
		case absent:
			if err := restoreClaimedFile(preClaim, target); err != nil {
				return err
			}
		case targetDigest == digestBytes(entry.Content):
			if err := os.Remove(preClaim); err != nil {
				return err
			}
		default:
			return fmt.Errorf("recovery target %s appeared while prestate and poststate were claimed", entry.Path)
		}
		return os.Remove(postClaim)
	}
	if postExists {
		if digest, digestErr := fileDigest(postClaim); digestErr != nil || digest != expectedPoststate {
			return fmt.Errorf("poststate claim for %s is invalid", entry.Path)
		}
		return publication.finishPoststateClaim(target, postClaim, entry)
	}
	if preExists {
		if !entry.Existed {
			return fmt.Errorf("unexpected prestate claim for new target %s", entry.Path)
		}
		if digest, digestErr := fileDigest(preClaim); digestErr != nil || digest != digestBytes(entry.Content) {
			return fmt.Errorf("prestate claim for %s is invalid", entry.Path)
		}
		targetDigest, absent, err := currentFileDigest(target)
		if err != nil {
			return err
		}
		switch {
		case absent:
			return restoreClaimedFile(preClaim, target)
		case targetDigest == digestBytes(entry.Content):
			return os.Remove(preClaim)
		case targetDigest == expectedPoststate:
			if err := os.Rename(target, postClaim); err != nil {
				return err
			}
			if err := restoreClaimedFile(preClaim, target); err != nil {
				return err
			}
			return os.Remove(postClaim)
		default:
			return fmt.Errorf("recovery target %s contains neither approved prestate nor poststate bytes", entry.Path)
		}
	}

	currentDigest, absent, err := currentFileDigest(target)
	if err != nil {
		return err
	}
	prestateDigest := ""
	if entry.Existed {
		prestateDigest = digestBytes(entry.Content)
	}
	if (entry.Existed && !absent && currentDigest == prestateDigest) || (!entry.Existed && absent) {
		return nil
	}
	poststateMatches := (expectedPoststate == "" && absent) ||
		(expectedPoststate != "" && !absent && currentDigest == expectedPoststate)
	if !poststateMatches {
		return fmt.Errorf("recovery target %s contains neither approved prestate nor poststate bytes", entry.Path)
	}
	if absent {
		return publication.writeNewExclusive(target, entry.Content, os.FileMode(entry.Mode))
	}
	claim, err := claimExpectedFile(target, expectedPoststate, poststateClaim)
	if err != nil {
		return err
	}
	return publication.finishPoststateClaim(target, claim, entry)
}

func (publication applicationPublicationOps) reconcilePublicationClaim(
	target string,
	entry journalEntry,
	expectedPoststate string,
) error {
	claim := claimPathForTarget(target, publicationClaim)
	exists, err := regularFileExists(claim)
	if err != nil || !exists {
		return err
	}
	claimDigest, err := fileDigest(claim)
	if err != nil {
		return err
	}
	prestateDigest := ""
	if entry.Existed {
		prestateDigest = digestBytes(entry.Content)
	}
	isPrestate := entry.Existed && claimDigest == prestateDigest
	isPoststate := expectedPoststate != "" && claimDigest == expectedPoststate
	targetDigest, absent, err := currentFileDigest(target)
	if err != nil {
		return err
	}
	if !isPrestate && !isPoststate {
		targetIsApproved := !absent &&
			((entry.Existed && targetDigest == prestateDigest) ||
				(expectedPoststate != "" && targetDigest == expectedPoststate))
		if absent || targetIsApproved {
			return os.Remove(claim)
		}
		return fmt.Errorf("publication claim for %s is incomplete while the recovery target has unapproved bytes", entry.Path)
	}
	switch {
	case absent && isPrestate:
		return restoreClaimedFile(claim, target)
	case absent:
		return os.Remove(claim)
	case targetDigest == claimDigest:
		return os.Remove(claim)
	default:
		return fmt.Errorf("recovery target %s changed while a publication claim was present", entry.Path)
	}
}

func regularFileExists(filePath string) (bool, error) {
	info, err := os.Lstat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("claim path %s is not a regular file", filePath)
	}
	return true, nil
}

func (publication applicationPublicationOps) finishPoststateClaim(target, postClaim string, entry journalEntry) error {
	currentDigest, absent, err := currentFileDigest(target)
	if err != nil {
		return err
	}
	if !absent {
		if entry.Existed && currentDigest == digestBytes(entry.Content) {
			return os.Remove(postClaim)
		}
		return fmt.Errorf("recovery target %s appeared while its poststate was claimed", entry.Path)
	}
	if entry.Existed {
		if err := publication.writeNewExclusive(target, entry.Content, os.FileMode(entry.Mode)); err != nil {
			return err
		}
	}
	return os.Remove(postClaim)
}

func currentFileDigest(target string) (string, bool, error) {
	digest, err := fileDigest(target)
	if errors.Is(err, os.ErrNotExist) {
		return "", true, nil
	}
	return digest, false, err
}
