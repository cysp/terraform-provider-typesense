package util

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func ImportStatePassthroughInt64ID(ctx context.Context, attrPath path.Path, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	val, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Resource Import Passthrough Invalid ID",
			err.Error(),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, attrPath, val)...)
}
