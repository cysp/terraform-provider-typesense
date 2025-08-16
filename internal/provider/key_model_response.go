package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/cysp/terraform-provider-typesense/internal/typesense-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (model *KeyModel) ReadFromResponse(ctx context.Context, apiKey *typesense.ApiKey) diag.Diagnostics {
	var diags diag.Diagnostics

	if keyID, keyIDOk := apiKey.ID.Get(); keyIDOk {
		model.ID = types.Int64Value(keyID)
	}

	model.Description = types.StringValue(apiKey.Description)

	model.Actions = util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, apiKey.Actions))(&diags)
	model.Collections = util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, apiKey.Collections))(&diags)

	if expiresAt, expiresAtOk := apiKey.ExpiresAt.Get(); expiresAtOk {
		model.ExpiresAt = types.Int64Value(expiresAt)
	} else {
		model.ExpiresAt = types.Int64Null()
	}

	if value, valueOk := apiKey.Value.Get(); valueOk {
		model.Value = types.StringValue(value)
		model.ValuePrefix = types.StringValue(value[:4])
	}

	if valuePrefix, valuePrefixOk := apiKey.ValuePrefix.Get(); valuePrefixOk {
		model.ValuePrefix = types.StringValue(valuePrefix)
	}

	return diags
}
