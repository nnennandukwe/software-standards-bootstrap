package prune

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Transition holds the review lock between preflight and a successful
// renderer/ADR write so invalid state cannot create an unrecorded artifact.
type Transition struct {
	review Review
	kind   string
	now    func() time.Time
	unlock func() error
	done   bool
}

// TransitionOptions identifies the review, event kind, and optional clock.
type TransitionOptions struct {
	ReviewID string
	Kind     string
	Now      func() time.Time
}

// BeginTransition validates and locks a rerender or ADR before its file write.
func BeginTransition(ctx context.Context, repoPath string, options TransitionOptions) (*Transition, error) {
	reviewID, kind, now := options.ReviewID, options.Kind, options.Now
	store, unlock, err := acquireReviewLock(ctx, repoPath, reviewID)
	if err != nil {
		return nil, err
	}
	review, diagnostics, err := loadReviewFromStore(ctx, store)
	if err != nil {
		return nil, errors.Join(err, unlock())
	}
	if len(diagnostics) != 0 {
		return nil, errors.Join(
			validationError("proposal has %d error(s)", len(diagnostics)),
			unlock(),
		)
	}
	if kind != EventRendered && kind != EventADR {
		return nil, errors.Join(fmt.Errorf("unsupported transition kind %s", kind), unlock())
	}
	applied := false
	verified := false
	for _, event := range review.Events {
		if event.Kind == EventApplied {
			applied = true
		}
		if event.Kind == EventVerified {
			verified = true
		}
		if event.Kind == kind {
			return nil, errors.Join(
				preconditionError("review already has a %s event", kind),
				unlock(),
			)
		}
	}
	if !applied {
		return nil, errors.Join(
			preconditionError("%s requires a completed application event", kind),
			unlock(),
		)
	}
	if kind == EventRendered && verified {
		return nil, errors.Join(
			preconditionError("rerendering must complete before verification; create a new review for additional rendered changes"),
			unlock(),
		)
	}
	return &Transition{review: review, kind: kind, now: now, unlock: unlock}, nil
}

// Complete appends the transition result and releases the review lock.
// After publishing files, a caller passes context.WithoutCancel(ctx) so event
// recording finishes in the same bounded sequence. A no-op keeps its caller
// context so cancellation can still stop the event write.
func (transition *Transition) Complete(ctx context.Context, payload any) (Event, error) {
	if transition.done {
		return Event{}, fmt.Errorf("transition is already complete")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	if transition.now == nil {
		transition.now = time.Now
	}
	event := newEvent(
		transition.review,
		transition.kind,
		data,
		transition.now().UTC(),
	)
	if err := appendEvent(ctx, transition.review, event); err != nil {
		return Event{}, err
	}
	transition.done = true
	if err := transition.unlock(); err != nil {
		return event, fmt.Errorf(
			"%w: transition event was recorded but lock cleanup failed: %w; run ssb prune recover --review %s --clear-stale-lock",
			ErrPrecondition,
			err,
			transition.review.Context.ReviewID,
		)
	}
	return event, nil
}

// Cancel releases a transition that did not write or whose write was rolled
// back.
func (transition *Transition) Cancel() error {
	if transition == nil || transition.done {
		return nil
	}
	transition.done = true
	return transition.unlock()
}
