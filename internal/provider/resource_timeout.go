package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func operationContext(ctx context.Context, timeout func(context.Context, time.Duration) (time.Duration, diag.Diagnostics), fallback time.Duration, diags *diag.Diagnostics) (context.Context, context.CancelFunc) {
	duration, timeoutDiags := timeout(ctx, fallback)
	diags.Append(timeoutDiags...)

	return context.WithTimeout(ctx, duration)
}
