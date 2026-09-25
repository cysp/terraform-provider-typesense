package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.ResourceWithValidateConfig = (*collectionResource)(nil)

func (r *collectionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var value, collectionSeparators, collectionSymbols types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("fields"), &value)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("token_separators"), &collectionSeparators)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("symbols_to_index"), &collectionSymbols)...)

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
		fieldPath := path.Root("fields").AtListIndex(index)

		validateFallbackFieldConfig(field, fieldPath, &resp.Diagnostics)
		warnFieldTokenOverrides(field, fieldPath, collectionSeparators, collectionSymbols, &resp.Diagnostics)

		if names[name] {
			resp.Diagnostics.AddAttributeError(fieldPath.AtName("name"), "Duplicate field declaration", "Declare each field name only once.")
		}

		names[name] = true
	}
}

func validateFallbackFieldConfig(field CollectionFieldModel, fieldPath path.Path, diags *diag.Diagnostics) {
	if field.Name.ValueString() != ".*" {
		return
	}

	for _, option := range []struct {
		name  string
		value attr.Value
	}{
		{"num_dim", field.NumDim},
		{"store", field.Store},
		{"range_index", field.RangeIndex},
		{"stem", field.Stem},
		{"stem_dictionary", field.StemDictionary},
		{"vec_dist", field.VecDist},
		{"token_separators", field.TokenSeparators},
		{"symbols_to_index", field.SymbolsToIndex},
	} {
		if !option.value.IsNull() {
			diags.AddAttributeError(fieldPath.AtName(option.name), "Unsupported fallback field option", "Typesense ignores this option on the exact .* fallback field. Remove it from this field's configuration.")
		}
	}
}

func warnFieldTokenOverrides(field CollectionFieldModel, fieldPath path.Path, collectionSeparators, collectionSymbols types.List, diags *diag.Diagnostics) {
	if field.Name.ValueString() == ".*" {
		return
	}

	for _, option := range []struct {
		name       string
		collection types.List
		field      types.List
	}{
		{"token_separators", collectionSeparators, field.TokenSeparators},
		{"symbols_to_index", collectionSymbols, field.SymbolsToIndex},
	} {
		if option.collection.IsNull() || option.collection.IsUnknown() || len(option.collection.Elements()) == 0 ||
			option.field.IsNull() || option.field.IsUnknown() || len(option.field.Elements()) == 0 {
			continue
		}

		diags.AddAttributeWarning(fieldPath.AtName(option.name), "Field tokenization overrides collection setting", "The field-level "+option.name+" list replaces the collection-level list for this field. Include any collection-level characters you also want in the field list.")
	}
}

// A configured field name may become known only after configuration validation.
// Recheck explicit fallback options against the resolved plan before mutation.
func validatePlannedFallbackFieldConfig(ctx context.Context, config tfsdk.Config, planned types.List) diag.Diagnostics {
	if config.Schema == nil {
		return nil
	}

	var configured types.List

	diags := config.GetAttribute(ctx, path.Root("fields"), &configured)

	if diags.HasError() || configured.IsNull() || configured.IsUnknown() || planned.IsNull() || planned.IsUnknown() {
		return diags
	}

	plannedElements := planned.Elements()
	for index, element := range configured.Elements() {
		if element.IsNull() || element.IsUnknown() || index >= len(plannedElements) || plannedElements[index].IsNull() || plannedElements[index].IsUnknown() {
			continue
		}

		name := plannedElements[index].(types.Object).Attributes()["name"].(types.String) //nolint:forcetypeassert // The field schema guarantees these types.
		if name.IsNull() || name.IsUnknown() || name.ValueString() != ".*" {
			continue
		}

		var field CollectionFieldModel
		diags.Append(element.(types.Object).As(ctx, &field, basetypes.ObjectAsOptions{})...) //nolint:forcetypeassert // The field schema guarantees object elements.

		if diags.HasError() {
			return diags
		}

		field.Name = name
		validateFallbackFieldConfig(field, path.Root("fields").AtListIndex(index), &diags)
	}

	return diags
}
