package refunds

import (
	"context"
	"fmt"
)

type Store interface {
	Begin(ctx context.Context, refundID, idempotencyKey string) error
	Complete(ctx context.Context, refundID string) error
}

type Service struct {
	store Store
}

func (s Service) Begin(ctx context.Context, refundID, idempotencyKey string) error {
	if err := s.store.Begin(ctx, refundID, idempotencyKey); err != nil {
		return fmt.Errorf("begin refund %s: %w", refundID, err)
	}
	return nil
}

func (s Service) Complete(ctx context.Context, refundID string) error {
	if err := s.store.Complete(ctx, refundID); err != nil {
		return fmt.Errorf("complete refund %s: %w", refundID, err)
	}
	return nil
}
