package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.ResourceWithValidateConfig = (*collectionResource)(nil)

func (r *collectionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var value types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("fields"), &value)...)

	if resp.Diagnostics.HasError() || value.IsNull() || value.IsUnknown() {
		return
	}

	names := make(map[string]bool, len(value.Elements()))
	for index, element := range value.Elements() {
		if element.IsNull() || element.IsUnknown() {
			continue
		}

		var field CollectionFieldModel
		resp.Diagnostics.Append(element.(types.Object).As(ctx, &field, basetypes.ObjectAsOptions{})...) //nolint:forcetypeassert // The framework schema guarantees object elements.

		if resp.Diagnostics.HasError() {
			return
		}

		if field.Name.IsUnknown() || field.Name.IsNull() {
			continue
		}

		name := field.Name.ValueString()
		if names[name] {
			resp.Diagnostics.AddAttributeError(path.Root("fields").AtListIndex(index).AtName("name"), "Duplicate field declaration", "Declare each field name only once.")
		}

		names[name] = true
	}
}
