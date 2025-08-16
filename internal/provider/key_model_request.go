package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/typesense-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func (model *KeyModel) ToAPIKeySchema(ctx context.Context) (typesense.ApiKeySchema, diag.Diagnostics) {
	var (
		apiKeySchema typesense.ApiKeySchema
		diags        diag.Diagnostics
	)

	apiKeySchema.Description = model.Description.ValueString()

	diags.Append(model.Actions.ElementsAs(ctx, &apiKeySchema.Actions, false)...)
	diags.Append(model.Collections.ElementsAs(ctx, &apiKeySchema.Collections, false)...)

	if !model.ExpiresAt.IsUnknown() && !model.ExpiresAt.IsNull() {
		apiKeySchema.ExpiresAt.SetTo(model.ExpiresAt.ValueInt64())
	}

	if !model.Value.IsUnknown() && !model.Value.IsNull() {
		apiKeySchema.Value.SetTo(model.Value.ValueString())
	}

	return apiKeySchema, diags
}
