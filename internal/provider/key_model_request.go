package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func (model *KeyModel) ToAPIKeySchema(ctx context.Context) (KeySchema, diag.Diagnostics) {
	var (
		apiKeySchema KeySchema
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

	if !model.Autodelete.IsUnknown() {
		apiKeySchema.Autodelete = model.Autodelete.ValueBoolPointer()
	}

	return apiKeySchema, diags
}
