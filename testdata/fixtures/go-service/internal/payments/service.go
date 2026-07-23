package payments

import (
	"context"
	"fmt"
)

type Store interface {
	Capture(ctx context.Context, paymentID, idempotencyKey string) error
}

type Service struct {
	store Store
}

func (s Service) Capture(ctx context.Context, paymentID, idempotencyKey string) error {
	if err := s.store.Capture(ctx, paymentID, idempotencyKey); err != nil {
		return fmt.Errorf("capture payment %s: %w", paymentID, err)
	}
	return nil
}
