//nolint:revive,stylecheck
package datasource_key

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	typesense_api "github.com/typesense/typesense-go/typesense/api"
)

func (model *KeyModel) ReadFromResponse(ctx context.Context, apiKey *typesense_api.ApiKey) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Id = types.Int64PointerValue(apiKey.Id)
	model.Actions = util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, apiKey.Actions))(&diags)
	model.Collections = util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, apiKey.Collections))(&diags)
	model.Description = types.StringValue(apiKey.Description)

	model.ExpiresAt = types.Int64PointerValue(apiKey.ExpiresAt)

	if apiKey.Value != nil {
		model.Value = types.StringPointerValue(apiKey.Value)
		model.ValuePrefix = types.StringValue((*apiKey.Value)[:4])
	}

	if apiKey.ValuePrefix != nil {
		model.ValuePrefix = types.StringPointerValue(apiKey.ValuePrefix)
	}

	return diags
}
