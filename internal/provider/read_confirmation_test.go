package provider //nolint:testpackage // Verify the shared not-found policy and cancellation classification.

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
)

func TestReadNotFoundConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("transient", func(t *testing.T) {
		t.Parallel()

		var calls int

		got, err := retrieveWithNotFoundConfirmation(t.Context(), time.Second, func(context.Context) (*int, error) {
			calls++
			if calls == 1 {
				return nil, &typesense.HTTPError{Status: http.StatusNotFound}
			}

			return new(7), nil
		})
		require.NoError(t, err)
		require.Equal(t, 7, *got)
		require.Equal(t, 2, calls)
	})
	t.Run("persistent", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32

		started := time.Now()
		got, err := retrieveWithNotFoundConfirmation(t.Context(), 100*time.Millisecond, func(context.Context) (*int, error) {
			calls.Add(1)

			return nil, &typesense.HTTPError{Status: http.StatusNotFound}
		})
		require.Nil(t, got)
		require.True(t, typesenseNotFound(err), "completed not-found observations remain distinguishable from read errors")
		require.GreaterOrEqual(t, calls.Load(), int32(2))
		require.GreaterOrEqual(t, time.Since(started), 100*time.Millisecond)
	})
	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		var calls int

		denied := &typesense.HTTPError{Status: http.StatusUnauthorized}
		got, err := retrieveWithNotFoundConfirmation(t.Context(), time.Second, func(context.Context) (*int, error) {
			calls++

			return nil, denied
		})
		require.Nil(t, got)
		require.ErrorIs(t, err, denied)
		require.Equal(t, 1, calls)
	})
	t.Run("caller_canceled_after_not_found", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		got, err := retrieveWithNotFoundConfirmation(ctx, time.Second, func(context.Context) (*int, error) {
			cancel()

			return nil, &typesense.HTTPError{Status: http.StatusNotFound}
		})
		require.Nil(t, got)
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, typesenseNotFound(err))
	})
	t.Run("caller_deadline", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		got, err := retrieveWithNotFoundConfirmation(ctx, time.Second, func(context.Context) (*int, error) {
			return nil, &typesense.HTTPError{Status: http.StatusNotFound}
		})
		require.Nil(t, got)
		require.Error(t, err)
		require.False(t, typesenseNotFound(err))
	})
	t.Run("ordinary_read_keeps_operation_deadline", func(t *testing.T) {
		t.Parallel()

		got, err := retrieveWithNotFoundConfirmation(t.Context(), 10*time.Millisecond, func(ctx context.Context) (*int, error) {
			select {
			case <-time.After(50 * time.Millisecond):
				return new(7), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		})
		require.NoError(t, err)
		require.Equal(t, 7, *got)
	})
	t.Run("inflight_retry_cannot_confirm_old_not_found", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		release := make(chan struct{})
		defer close(release)

		got, err := retrieveWithNotFoundConfirmation(ctx, 50*time.Millisecond, func(ctx context.Context) (*int, error) {
			if calls.Add(1) == 1 {
				return nil, &typesense.HTTPError{Status: http.StatusNotFound}
			}

			<-ctx.Done()
			<-release

			return nil, ctx.Err()
		})
		require.Nil(t, got)
		require.Error(t, err)
		require.False(t, typesenseNotFound(err), "a canceled in-flight read is not evidence of deletion")
		require.EqualValues(t, 2, calls.Load())
	})
}
