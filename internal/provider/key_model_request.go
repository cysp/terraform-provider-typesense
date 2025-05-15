package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	typesense_api "github.com/typesense/typesense-go/typesense/api"
)

func (model *KeyModel) ToAPIKeySchema(ctx context.Context) (typesense_api.ApiKeySchema, diag.Diagnostics) {
	var (
		apiKeySchema typesense_api.ApiKeySchema
		diags        diag.Diagnostics
	)

	apiKeySchema.Description = model.Description.ValueString()

	diags.Append(model.Actions.ElementsAs(ctx, &apiKeySchema.Actions, false)...)
	diags.Append(model.Collections.ElementsAs(ctx, &apiKeySchema.Collections, false)...)

	if !model.ExpiresAt.IsUnknown() {
		apiKeySchema.ExpiresAt = model.ExpiresAt.ValueInt64Pointer()
	}

	if !model.Value.IsUnknown() {
		apiKeySchema.Value = model.Value.ValueStringPointer()
	}

	return apiKeySchema, diags
}
