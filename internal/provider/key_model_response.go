package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const keyPrefixLength = 4

func (model *KeyModel) ReadFromResponse(ctx context.Context, apiKey *KeyResponse) diag.Diagnostics {
	metadata, diags := keyMetadataFromResponse(ctx, apiKey)
	model.ID = metadata.ID
	model.Description = metadata.Description
	model.Actions = metadata.Actions
	model.Collections = metadata.Collections
	model.Autodelete = metadata.Autodelete
	model.ExpiresAt = metadata.ExpiresAt

	if apiKey.Value != nil {
		model.Value = types.StringPointerValue(apiKey.Value)
		model.ValuePrefix = types.StringValue((*apiKey.Value)[:min(keyPrefixLength, len(*apiKey.Value))])
	}

	if !metadata.ValuePrefix.IsNull() {
		model.ValuePrefix = metadata.ValuePrefix
	}

	return diags
}

func keyMetadataFromResponse(ctx context.Context, key *KeyResponse) (KeyDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	actions := util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, key.Actions))(&diags)
	collections := util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, key.Collections))(&diags)

	return KeyDataSourceModel{
		ID:          types.Int64PointerValue(key.Id),
		Description: types.StringValue(key.Description),
		Actions:     actions,
		Collections: collections,
		Autodelete:  types.BoolPointerValue(key.Autodelete),
		ExpiresAt:   types.Int64PointerValue(key.ExpiresAt),
		ValuePrefix: types.StringPointerValue(key.ValuePrefix),
	}, diags
}
