package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const keyPrefixLength = 4

func (model *KeyModel) ReadFromResponse(ctx context.Context, apiKey *KeyResponse) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.Int64PointerValue(apiKey.Id)
	model.Description = types.StringValue(apiKey.Description)

	model.Actions = util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, apiKey.Actions))(&diags)
	model.Collections = util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, apiKey.Collections))(&diags)

	model.Autodelete = types.BoolPointerValue(apiKey.Autodelete)

	model.ExpiresAt = types.Int64PointerValue(apiKey.ExpiresAt)

	if apiKey.Value != nil {
		model.Value = types.StringPointerValue(apiKey.Value)
		model.ValuePrefix = types.StringValue((*apiKey.Value)[:min(keyPrefixLength, len(*apiKey.Value))])
	}

	if apiKey.ValuePrefix != nil {
		model.ValuePrefix = types.StringPointerValue(apiKey.ValuePrefix)
	}

	return diags
}
