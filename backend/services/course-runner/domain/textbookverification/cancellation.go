package textbookverification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrVerificationCancelled means a confirmed user cancellation won the race.
var ErrVerificationCancelled = errors.New("textbook verification cancelled")

// ErrVerificationTaskTerminal acknowledges an already completed delivery.
var ErrVerificationTaskTerminal = errors.New("textbook verification task is terminal")

// observeCancellation also aborts execution when cancellation state cannot be
// read. Losing the task store is never permission to continue consuming code.
func (consumer *Consumer) observeCancellation(ctx context.Context, cancel context.CancelCauseFunc, taskID uuid.UUID) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			queryCtx, finish := context.WithTimeout(ctx, 2*time.Second)
			cancelled, err := consumer.tasks.IsCancelled(queryCtx, taskID)
			finish()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				cancel(fmt.Errorf("read verification cancellation: %w", err))
				return
			}
			if cancelled {
				cancel(ErrVerificationCancelled)
				return
			}
		}
	}
}
