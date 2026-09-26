package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

var _ resource.ResourceWithValidateConfig = (*collectionResource)(nil)

const (
	collectionReservedIDMessage      = "Typesense manages the document id field automatically and omits it from collection schemas. Remove the id declaration from fields."
	collectionDynamicOptionalMessage = "Typesense requires non-nested dynamic fields to set optional = true. With enable_nested_fields = true, object/object[] fields and dotted names without .* are nested and may set optional = false."
	collectionUnknownFallbackMessage = "The fallback field was unknown during planning, so the provider cannot tell whether options ignored by Typesense were explicitly configured on the exact .* fallback. Declare the fallback directly in fields, or make its value known during planning."
)

func (r *collectionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var value, collectionSeparators, collectionSymbols types.List

	var enableNested types.Bool

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("fields"), &value)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("token_separators"), &collectionSeparators)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("symbols_to_index"), &collectionSymbols)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("enable_nested_fields"), &enableNested)...)

	if resp.Diagnostics.HasError() || value.IsNull() || value.IsUnknown() {
		return
	}

	names := make(map[string][]types.String, len(value.Elements()))
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
		validateCollectionFieldConfig(field, fieldPath, enableNested, collectionSeparators, collectionSymbols, &resp.Diagnostics)

		previous := names[name]
		if len(previous) > 1 || len(previous) == 1 && !previous[0].IsUnknown() && !field.Type.IsUnknown() &&
			!collectionSameNamePairAllowed(name, previous[0].ValueString(), field.Type.ValueString()) {
			resp.Diagnostics.AddAttributeError(fieldPath.AtName("name"), "Duplicate field declaration", "A field name may appear twice only for one named auto or string* declaration and one concrete field of another type.")
		}

		names[name] = append(previous, field.Type)
	}
}

func validateCollectionFieldConfig(field CollectionFieldModel, fieldPath path.Path, enableNested types.Bool, collectionSeparators, collectionSymbols types.List, diags *diag.Diagnostics) {
	name := field.Name.ValueString()
	if name == "id" {
		diags.AddAttributeError(fieldPath.AtName("name"), "Reserved field name", collectionReservedIDMessage)
	}

	validateFallbackFieldConfig(field, fieldPath, diags)

	if name != ".*" && collectionFieldRequiredDynamic(field) {
		canNest := collectionDynamicFieldCanNest(name, field.Type.ValueString())
		if !canNest || (!enableNested.IsNull() && !enableNested.IsUnknown() && !enableNested.ValueBool()) {
			diags.AddAttributeError(fieldPath.AtName("optional"), "Invalid dynamic field optional", collectionDynamicOptionalMessage)
		}
	}

	warnFieldTokenOverrides(field, fieldPath, collectionSeparators, collectionSymbols, diags)
}

func collectionFieldRequiredDynamic(field CollectionFieldModel) bool {
	return field.Optional.Equal(types.BoolValue(false)) && !field.Name.IsNull() && !field.Name.IsUnknown() &&
		!field.Type.IsNull() && !field.Type.IsUnknown() &&
		isCollectionDynamicField(api.Field{Name: field.Name.ValueString(), Type: field.Type.ValueString()})
}

func collectionDynamicFieldCanNest(name, fieldType string) bool {
	return fieldType == "object" || fieldType == "object[]" || (strings.Contains(name, ".") && !strings.Contains(name, ".*"))
}

func validatePlannedDynamicOptional(ctx context.Context, planned CollectionModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if planned.Fields.IsNull() || planned.Fields.IsUnknown() {
		return diags
	}

	var fields []CollectionFieldModel
	diags.Append(planned.Fields.ElementsAs(ctx, &fields, false)...)

	if diags.HasError() {
		return diags
	}

	for index, field := range fields {
		if field.Name.ValueString() != ".*" && collectionFieldRequiredDynamic(field) &&
			(!planned.EnableNestedFields.ValueBool() || !collectionDynamicFieldCanNest(field.Name.ValueString(), field.Type.ValueString())) {
			diags.AddAttributeError(path.Root("fields").AtListIndex(index).AtName("optional"), "Invalid dynamic field optional", collectionDynamicOptionalMessage)
		}
	}

	return diags
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

	for _, option := range []struct {
		name    string
		invalid bool
	}{
		{"optional", !field.Optional.IsNull() && !field.Optional.IsUnknown() && !field.Optional.ValueBool()},
		{"facet", !field.Facet.IsNull() && !field.Facet.IsUnknown() && field.Facet.ValueBool()},
		{"index", !field.Index.IsNull() && !field.Index.IsUnknown() && !field.Index.ValueBool()},
		{"reference", !field.Reference.IsNull() && !field.Reference.IsUnknown() && field.Reference.ValueString() != ""},
	} {
		if option.invalid {
			diags.AddAttributeError(fieldPath.AtName(option.name), "Invalid fallback field option", "Typesense requires the exact .* fallback field to be optional and indexed, and does not allow faceting or references on it.")
		}
	}
}

func warnFieldTokenOverrides(field CollectionFieldModel, fieldPath path.Path, collectionSeparators, collectionSymbols types.List, diags *diag.Diagnostics) {
	if field.Name.ValueString() == ".*" {
		return
	}

	for _, option := range fieldTokenOverrideOptions(field, collectionSeparators, collectionSymbols) {
		addFieldTokenOverrideWarning(fieldPath.AtName(option), option, diags)
	}
}

func fieldTokenOverrideOptions(field CollectionFieldModel, collectionSeparators, collectionSymbols types.List) []string {
	var overrides []string

	for _, option := range []struct {
		name       string
		collection types.List
		field      types.List
	}{
		{"token_separators", collectionSeparators, field.TokenSeparators},
		{"symbols_to_index", collectionSymbols, field.SymbolsToIndex},
	} {
		if !fieldTokenListDropsCollectionValues(option.collection, option.field) {
			continue
		}

		overrides = append(overrides, option.name)
	}

	return overrides
}

func addFieldTokenOverrideWarning(optionPath path.Path, option string, diags *diag.Diagnostics) {
	diags.AddAttributeWarning(optionPath, "Field tokenization overrides collection setting", "The field-level "+option+" list replaces the collection-level list for this field. Include any collection-level characters you also want in the field list.")
}

func fieldTokenListDropsCollectionValues(collection, field types.List) bool {
	if collection.IsNull() || collection.IsUnknown() || field.IsNull() || field.IsUnknown() ||
		len(collection.Elements()) == 0 || len(field.Elements()) == 0 {
		return false
	}

	for _, value := range collection.Elements() {
		if value.IsUnknown() {
			return false
		}
	}

	for _, value := range field.Elements() {
		if value.IsUnknown() {
			return false
		}
	}

	for _, value := range collection.Elements() {
		if !slices.ContainsFunc(field.Elements(), value.Equal) {
			return true
		}
	}

	return false
}

// A plan may contain tokenization values that were unknown in configuration.
func collectionTokenOverridePaths(ctx context.Context, planned CollectionModel) (map[string]path.Path, diag.Diagnostics) {
	var diags diag.Diagnostics

	paths := make(map[string]path.Path)
	if planned.Fields.IsNull() || planned.Fields.IsUnknown() {
		return paths, diags
	}

	var fields []CollectionFieldModel
	diags.Append(planned.Fields.ElementsAs(ctx, &fields, false)...)

	if diags.HasError() {
		return paths, diags
	}

	for index, field := range fields {
		if field.Name.IsNull() || field.Name.IsUnknown() || field.Name.ValueString() == ".*" {
			continue
		}

		for _, option := range fieldTokenOverrideOptions(field, planned.TokenSeparators, planned.SymbolsToIndex) {
			paths[fmt.Sprintf("%d/%s", index, option)] = path.Root("fields").AtListIndex(index).AtName(option)
		}
	}

	return paths, diags
}

// Configuration validation reports known overrides during planning. Report only
// those that became known at apply, without repeating the earlier warning.
func warnResolvedFieldTokenOverrides(ctx context.Context, config tfsdk.Config, planned CollectionModel) diag.Diagnostics {
	if config.Schema == nil {
		return nil
	}

	var configured CollectionModel

	diags := config.Get(ctx, &configured)
	if diags.HasError() {
		return diags
	}

	known, knownDiags := collectionTokenOverridePaths(ctx, configured)
	diags.Append(knownDiags...)

	resolved, resolvedDiags := collectionTokenOverridePaths(ctx, planned)
	diags.Append(resolvedDiags...)

	if diags.HasError() {
		return diags
	}

	for key, optionPath := range resolved {
		if _, alreadyWarned := known[key]; alreadyWarned {
			continue
		}

		option := "token_separators"
		if strings.HasSuffix(key, "/symbols_to_index") {
			option = "symbols_to_index"
		}

		addFieldTokenOverrideWarning(optionPath, option, &diags)
	}

	return diags
}

// A configured field name may become known only after configuration validation.
// Recheck explicit fallback options against the resolved plan before mutation.
func validatePlannedFallbackFieldConfig(ctx context.Context, config tfsdk.Config, planned types.List) diag.Diagnostics {
	if config.Schema == nil {
		return nil
	}

	var configured types.List

	diags := config.GetAttribute(ctx, path.Root("fields"), &configured)

	if diags.HasError() || configured.IsNull() || planned.IsNull() || planned.IsUnknown() {
		return diags
	}

	if configured.IsUnknown() {
		if _, found := collectionExactFallback(planned); found {
			diags.AddAttributeError(path.Root("fields"), "Cannot validate computed fallback fields", collectionUnknownFallbackMessage)
		}

		return diags
	}

	plannedElements := planned.Elements()
	for index, element := range configured.Elements() {
		if index >= len(plannedElements) || plannedElements[index].IsNull() || plannedElements[index].IsUnknown() {
			continue
		}

		name := plannedElements[index].(types.Object).Attributes()["name"].(types.String) //nolint:forcetypeassert // The field schema guarantees these types.
		if name.IsNull() || name.IsUnknown() || name.ValueString() != ".*" {
			continue
		}

		if element.IsUnknown() {
			diags.AddAttributeError(path.Root("fields").AtListIndex(index), "Cannot validate computed fallback field", collectionUnknownFallbackMessage)

			continue
		}

		if element.IsNull() {
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
