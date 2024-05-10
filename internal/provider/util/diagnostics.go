package util

import "github.com/hashicorp/terraform-plugin-framework/diag"

func DiagnosticsAppender[T any](v T, d diag.Diagnostics) func(*diag.Diagnostics) T {
	return func(ds *diag.Diagnostics) T {
		ds.Append(d...)

		return v
	}
}
