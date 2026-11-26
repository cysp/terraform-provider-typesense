package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.ResourceWithValidateConfig = (*collectionResource)(nil)

func (r *collectionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var (
		value   types.List
		sorting types.String
	)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("default_sorting_field"), &sorting)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("fields"), &value)...)

	if resp.Diagnostics.HasError() || value.IsNull() || value.IsUnknown() {
		return
	}

	allNamesKnown := true

	names := make(map[string]bool, len(value.Elements()))
	for index, element := range value.Elements() {
		if element.IsNull() || element.IsUnknown() {
			allNamesKnown = false

			continue
		}

		var field CollectionFieldModel
		resp.Diagnostics.Append(element.(types.Object).As(ctx, &field, basetypes.ObjectAsOptions{})...) //nolint:forcetypeassert // The framework schema guarantees object elements.

		if resp.Diagnostics.HasError() {
			return
		}

		if field.Name.IsUnknown() || field.Name.IsNull() {
			allNamesKnown = false

			continue
		}

		name := field.Name.ValueString()
		if names[name] {
			resp.Diagnostics.AddAttributeError(path.Root("fields").AtListIndex(index).AtName("name"), "Duplicate field declaration", "Declare each field name only once.")
		}

		names[name] = true
	}

	if name := sorting.ValueString(); allNamesKnown && name != "" && !names[name] {
		resp.Diagnostics.AddAttributeError(path.Root("default_sorting_field"), "Missing default sorting field", fmt.Sprintf("default_sorting_field names %q, which is absent from fields. Keep the field declaration or remove default_sorting_field from the configuration.", name))
	}
}
