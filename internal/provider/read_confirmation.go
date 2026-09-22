package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

const (
	notFoundConfirmationTimeout = 5 * time.Second
	notFoundPollInterval        = 500 * time.Millisecond
)

type readObservation[T any] struct {
	value *T
	err   error
}

// Replicas can briefly report missing resources after a successful write.
// Only a completed not-found read after the confirmation period proves absence
// under this bounded policy; interrupted reads must retain resource state.
func retrieveWithNotFoundConfirmation[T any](ctx context.Context, timeout time.Duration, retrieve func(context.Context) (*T, error)) (*T, error) {
	current, err := retrieve(ctx)
	if ctx.Err() != nil {
		return nil, fmt.Errorf("reading resource: %w", ctx.Err())
	}

	if err == nil {
		return current, nil
	}

	if !typesenseNotFound(err) {
		return nil, fmt.Errorf("reading resource: %w", err)
	}

	confirmAfter := time.Now().Add(timeout)

	remaining := defaultOperationTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining = time.Until(deadline)
	}

	waiter := retry.StateChangeConf{
		Pending:      []string{"pending"},
		Target:       []string{"present", "absent"},
		Timeout:      remaining,
		PollInterval: notFoundPollInterval,
		Refresh: func() (any, string, error) {
			observed, readErr := retrieve(ctx)
			if readErr == nil {
				return &readObservation[T]{value: observed}, "present", nil
			}

			if !typesenseNotFound(readErr) {
				return nil, "", fmt.Errorf("reading resource: %w", readErr)
			}

			observation := &readObservation[T]{err: readErr}
			if time.Now().Before(confirmAfter) {
				return observation, "pending", nil
			}

			return observation, "absent", nil
		},
	}

	result, err := waiter.WaitForStateContext(ctx)
	if ctx.Err() != nil {
		return nil, fmt.Errorf("confirming resource existence: %w", ctx.Err())
	}

	if err != nil {
		return nil, fmt.Errorf("confirming resource existence: %w", err)
	}

	// Refresh only reaches a target with a completed observation.
	observation := result.(*readObservation[T]) //nolint:forcetypeassert
	if observation.err != nil {
		return nil, fmt.Errorf("confirming resource existence: %w", observation.err)
	}

	return observation.value, nil
}
