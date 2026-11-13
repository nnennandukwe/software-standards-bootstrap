package prune

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	EventApproved = "approved"
	EventApplied  = "applied"
	EventRendered = "rendered"
	EventADR      = "adr-recorded"
	EventVerified = "verified"
)

// Event is one digest-chained review lifecycle transition.
type Event struct {
	Schema              string          `json:"schema"`
	ID                  string          `json:"id"`
	ReviewID            string          `json:"review_id"`
	Kind                string          `json:"kind"`
	RecordedAt          string          `json:"recorded_at"`
	BaselineCommit      string          `json:"baseline_commit"`
	ContextDigest       string          `json:"context_digest"`
	ProposalDigest      string          `json:"proposal_digest"`
	PreviousEventDigest string          `json:"previous_event_digest,omitempty"`
	Payload             json.RawMessage `json:"payload"`
	EventDigest         string          `json:"event_digest"`
}

type renderedEventPayload struct {
	Path          string                  `json:"path"`
	Changed       bool                    `json:"changed"`
	DryRun        bool                    `json:"dry_run"`
	Exists        bool                    `json:"exists"`
	SourceDigest  string                  `json:"source_digest"`
	ContentDigest string                  `json:"content_digest"`
	OutputDigest  string                  `json:"output_digest"`
	Routing       *renderedRoutingPayload `json:"routing,omitempty"`
}

type renderedRoutingPayload struct {
	Path       string `json:"path"`
	Changed    bool   `json:"changed"`
	Exists     bool   `json:"exists"`
	TreeDigest string `json:"tree_digest"`
}

type adrEventPayload struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
	DryRun  bool   `json:"dry_run"`
}

func validateReviewEvents(ctx context.Context, review Review) error {
	seen := make(map[string]bool)
	applied := false
	rendered := false
	var approval ApprovalPayload
	var appliedResult ApplyResult
	applicationEventDigest := ""
	renderEventDigest := ""
	for index, event := range review.Events {
		if event.ReviewID != review.Context.ReviewID ||
			event.BaselineCommit != review.Context.BaselineCommit ||
			event.ContextDigest != review.Context.ContextDigest ||
			event.ProposalDigest != review.ProposalDigest {
			return fmt.Errorf("event %d is bound to a different review identity", index+1)
		}
		if event.ID != fmt.Sprintf("%s-%03d", event.Kind, index+1) {
			return fmt.Errorf("event %d has an invalid id", index+1)
		}
		if _, err := time.Parse(time.RFC3339, event.RecordedAt); err != nil {
			return fmt.Errorf("event %d has an invalid recorded_at", index+1)
		}
		if seen[event.Kind] {
			return fmt.Errorf("event kind %s appears more than once", event.Kind)
		}
		switch event.Kind {
		case EventApproved:
			if index != 0 {
				return fmt.Errorf("approval must be the first event")
			}
			var payload ApprovalPayload
			if err := decodeStrictJSON(event.Payload, &payload); err != nil {
				return fmt.Errorf("approval payload is invalid: %w", err)
			}
			if err := validateRecordedApproval(review.Proposal, payload); err != nil {
				return fmt.Errorf("approval payload is invalid: %w", err)
			}
			approval = payload
		case EventApplied:
			if !seen[EventApproved] {
				return fmt.Errorf("application event precedes approval")
			}
			var payload ApplyResult
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				payload.DryRun || payload.NoChangesApproved || !validDigest(payload.PlanDigest) ||
				len(payload.Changes) == 0 {
				return fmt.Errorf("application payload is invalid")
			}
			expected, err := canonicalApplicationPlan(ctx, review, approval)
			if err != nil {
				return fmt.Errorf("derive application plan: %w", err)
			}
			actualDigest, _ := canonicalDigest(payload.Changes)
			expectedDigest, _ := canonicalDigest(expected.Changes)
			if payload.PlanDigest != expected.PlanDigest || actualDigest != expectedDigest {
				return fmt.Errorf("application payload does not match approved changes")
			}
			applied = true
			appliedResult = payload
			applicationEventDigest = event.EventDigest
		case EventRendered:
			if !applied {
				return fmt.Errorf("rerender event precedes application")
			}
			if seen[EventVerified] {
				return fmt.Errorf("rerender event follows verification")
			}
			var payload renderedEventPayload
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				payload.Path != "AGENTS.md" || payload.DryRun ||
				!validDigest(payload.SourceDigest) || !validDigest(payload.ContentDigest) ||
				!validDigest(payload.OutputDigest) || !validRenderedRoutingPayload(payload.Routing) {
				return fmt.Errorf("rerender payload is invalid")
			}
			rendered = true
			renderEventDigest = event.EventDigest
		case EventADR:
			if !applied {
				return fmt.Errorf("ADR event precedes application")
			}
			var payload adrEventPayload
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				!payload.Created || payload.DryRun || !safeRelativePath(payload.Path) ||
				!strings.HasSuffix(payload.Path, ".md") {
				return fmt.Errorf("ADR payload is invalid")
			}
		case EventVerified:
			if !applied {
				return fmt.Errorf("verification event precedes application")
			}
			var payload VerificationResult
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				len(payload.Receipts) == 0 ||
				payload.ApplicationEventDigest != applicationEventDigest ||
				payload.PlanDigest != appliedResult.PlanDigest ||
				payload.RenderEventDigest != renderEventDigest {
				return fmt.Errorf("verification payload is invalid")
			}
			required := requiredRecordedChecks(review.Proposal, approval)
			if len(payload.Receipts) != len(required) {
				return fmt.Errorf("verification payload does not cover every required check")
			}
			for id := range required {
				if !validDigest(payload.Receipts[id]) {
					return fmt.Errorf("verification payload lacks a valid receipt for %s", id)
				}
			}
			if changesRules(review) && !rendered {
				return fmt.Errorf("verification event precedes required rerendering")
			}
		default:
			return fmt.Errorf("event %d has unsupported kind %q", index+1, event.Kind)
		}
		seen[event.Kind] = true
	}
	return nil
}

func validRenderedRoutingPayload(payload *renderedRoutingPayload) bool {
	return payload == nil ||
		payload.Path == ".software-standards/routing" && validDigest(payload.TreeDigest)
}

func loadEvents(reader io.Reader) ([]Event, error) {
	events := make([]Event, 0)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), 64<<20)
	previous := ""
	for scanner.Scan() {
		var event Event
		if err := decodeStrictJSON(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("parse review event %d: %w", len(events)+1, err)
		}
		if event.Schema != EventSchema || event.PreviousEventDigest != previous {
			return nil, fmt.Errorf("review event chain is invalid at event %d", len(events)+1)
		}
		expected := event.EventDigest
		event.EventDigest = ""
		actual, err := canonicalDigest(event)
		if err != nil || expected != actual {
			return nil, fmt.Errorf("review event digest is invalid at event %d", len(events)+1)
		}
		event.EventDigest = expected
		previous = expected
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read review events: %w", err)
	}
	return events, nil
}

func newEvent(review Review, kind string, payload []byte, recordedAt time.Time) Event {
	previous := ""
	if len(review.Events) != 0 {
		previous = review.Events[len(review.Events)-1].EventDigest
	}
	event := Event{
		Schema:              EventSchema,
		ID:                  fmt.Sprintf("%s-%03d", kind, len(review.Events)+1),
		ReviewID:            review.Context.ReviewID,
		Kind:                kind,
		RecordedAt:          recordedAt.Format(time.RFC3339),
		BaselineCommit:      review.Context.BaselineCommit,
		ContextDigest:       review.Context.ContextDigest,
		ProposalDigest:      review.ProposalDigest,
		PreviousEventDigest: previous,
		Payload:             payload,
	}
	event.EventDigest, _ = canonicalDigest(event)
	return event
}

func appendEvent(ctx context.Context, review Review, event Event) error {
	store := review.store
	owned := false
	if store == nil {
		var err error
		store, err = openReviewStore(review.RepoRoot, review.Context.ReviewID)
		if err != nil {
			return fmt.Errorf("open anchored review event path: %w", err)
		}
		owned = true
	}
	if owned {
		defer store.Close()
	}
	if store.absolute != review.Root {
		return fmt.Errorf("review event path changed after loading")
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode review event: %w", err)
	}
	data = append(data, '\n')
	existing, _, err := store.ReadRegular("events.jsonl")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read review event log before append: %w", err)
	}
	if len(existing) != 0 && existing[len(existing)-1] != '\n' {
		return fmt.Errorf("review event log does not end at a complete record")
	}
	next := make([]byte, 0, len(existing)+len(data))
	next = append(next, existing...)
	next = append(next, data...)
	if err := ctx.Err(); err != nil {
		return err
	}
	// Atomic replacement and its cleanup complete once started.
	if err := store.events.write(store, next, 0o644); err != nil {
		return fmt.Errorf("atomically replace review event log: %w", err)
	}
	return nil
}

type reviewEventOps struct {
	write      func(*reviewStore, []byte, os.FileMode) error
	removeLock func(*reviewStore) error
}

func defaultReviewEventOps() reviewEventOps {
	return reviewEventOps{
		write: func(store *reviewStore, data []byte, mode os.FileMode) error {
			return store.AtomicWrite("events.jsonl", data, mode)
		},
		removeLock: func(store *reviewStore) error { return store.Remove(".transition.lock") },
	}
}
