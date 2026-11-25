package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

const (
	collectionVerificationInterval = 500 * time.Millisecond
	collectionVerificationMatches  = 2
)

func (r *collectionResource) waitForCollectionUpdate(ctx context.Context, desired api.CollectionSchema) (*api.CollectionResponse, error) {
	deadline, _ := ctx.Deadline()
	waiter := retry.StateChangeConf{
		Pending:                   []string{"pending"},
		Target:                    []string{"converged"},
		Timeout:                   time.Until(deadline),
		PollInterval:              collectionVerificationInterval,
		ContinuousTargetOccurence: collectionVerificationMatches,
		Refresh: func() (any, string, error) {
			current, err := r.providerData.api.getCollection(ctx, desired.Name)
			if err != nil {
				return nil, "", fmt.Errorf("reading collection: %w", err)
			}

			if !sameCollectionFields(current.Fields, desired.Fields) {
				return current, "pending", nil
			}

			return current, "converged", nil
		},
	}

	result, err := waiter.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for collection schema convergence: %w", err)
	}

	// Refresh only returns a CollectionResponse when the target is reached.
	return result.(*api.CollectionResponse), nil //nolint:forcetypeassert
}
