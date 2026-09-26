package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func (model *CollectionModel) ToAPICollectionSchema(ctx context.Context) (typesense_api.CollectionSchema, diag.Diagnostics) {
	var (
		collectionSchema typesense_api.CollectionSchema
		diags            diag.Diagnostics
		fields           []CollectionFieldModel
	)

	collectionSchema.Name = model.Name.ValueString()

	diags.Append(model.Fields.ElementsAs(ctx, &fields, false)...)

	if diags.HasError() {
		return collectionSchema, diags
	}

	seen := make(map[string][]string, len(fields))
	for _, field := range fields {
		if field.Name.IsUnknown() || field.Name.IsNull() || field.Type.IsUnknown() || field.Type.IsNull() {
			continue
		}

		name, fieldType := field.Name.ValueString(), field.Type.ValueString()
		previous := seen[name]

		if len(previous) > 1 || len(previous) == 1 && !collectionSameNamePairAllowed(name, previous[0], fieldType) {
			diags.AddError("Duplicate field declaration", fmt.Sprintf("Field %q may appear twice only for one named auto or string* declaration and one concrete field of another type.", name))
		}

		seen[name] = append(previous, fieldType)
	}

	if diags.HasError() {
		return collectionSchema, diags
	}

	for _, field := range fields {
		apiField, fieldDiags := field.ToAPIField(ctx)
		diags.Append(fieldDiags...)

		collectionSchema.Fields = append(collectionSchema.Fields, apiField)
	}

	if !model.DefaultSortingField.IsUnknown() && !model.DefaultSortingField.IsNull() {
		collectionSchema.DefaultSortingField = model.DefaultSortingField.ValueStringPointer()
	}

	if !model.EnableNestedFields.IsUnknown() && !model.EnableNestedFields.IsNull() {
		collectionSchema.EnableNestedFields = model.EnableNestedFields.ValueBoolPointer()
	}

	if !model.SymbolsToIndex.IsUnknown() && !model.SymbolsToIndex.IsNull() {
		var symbolsToIndex []string
		diags.Append(model.SymbolsToIndex.ElementsAs(ctx, &symbolsToIndex, false)...)
		collectionSchema.SymbolsToIndex = &symbolsToIndex
	}

	if !model.TokenSeparators.IsUnknown() && !model.TokenSeparators.IsNull() {
		var tokenSeparators []string
		diags.Append(model.TokenSeparators.ElementsAs(ctx, &tokenSeparators, false)...)
		collectionSchema.TokenSeparators = &tokenSeparators
	}

	return collectionSchema, diags
}

func (model *CollectionFieldModel) ToAPIField(ctx context.Context) (typesense_api.Field, diag.Diagnostics) {
	var diags diag.Diagnostics

	if model.NumDim.ValueInt64() > int64(int(^uint(0)>>1)) {
		diags.AddError("Invalid field num_dim", fmt.Sprintf("num_dim %d overflows int", model.NumDim.ValueInt64()))

		return typesense_api.Field{}, diags
	}

	apiField := typesense_api.Field{
		Name: model.Name.ValueString(),
		Type: model.Type.ValueString(),
	}

	diags.Append(model.validateCollectionFieldOptions()...)

	if diags.HasError() {
		return typesense_api.Field{}, diags
	}

	if !model.Facet.IsUnknown() && !model.Facet.IsNull() {
		apiField.Facet = model.Facet.ValueBoolPointer()
	}

	if !model.Index.IsUnknown() && !model.Index.IsNull() {
		apiField.Index = model.Index.ValueBoolPointer()
	}

	if !model.Infix.IsUnknown() && !model.Infix.IsNull() {
		apiField.Infix = model.Infix.ValueBoolPointer()
	}

	if !model.Locale.IsUnknown() && !model.Locale.IsNull() {
		apiField.Locale = model.Locale.ValueStringPointer()
	}

	if !model.NumDim.IsUnknown() && !model.NumDim.IsNull() {
		numDim := int(model.NumDim.ValueInt64())
		apiField.NumDim = &numDim
	}

	if !model.Optional.IsUnknown() && !model.Optional.IsNull() {
		apiField.Optional = model.Optional.ValueBoolPointer()
	}

	if !model.Reference.IsUnknown() && !model.Reference.IsNull() {
		apiField.Reference = model.Reference.ValueStringPointer()
	}

	if !model.Sort.IsUnknown() && !model.Sort.IsNull() {
		apiField.Sort = model.Sort.ValueBoolPointer()
	}

	diags.Append(model.setAdditionalAPIFieldOptions(ctx, &apiField)...)

	return apiField, diags
}

func (model *CollectionFieldModel) setAdditionalAPIFieldOptions(ctx context.Context, apiField *typesense_api.Field) diag.Diagnostics {
	var diags diag.Diagnostics

	if !model.Store.IsUnknown() && !model.Store.IsNull() {
		apiField.Store = model.Store.ValueBoolPointer()
	}

	if !model.RangeIndex.IsUnknown() && !model.RangeIndex.IsNull() {
		apiField.RangeIndex = model.RangeIndex.ValueBoolPointer()
	}

	if !model.Stem.IsUnknown() && !model.Stem.IsNull() {
		apiField.Stem = model.Stem.ValueBoolPointer()
	}

	if !model.StemDictionary.IsUnknown() && !model.StemDictionary.IsNull() {
		apiField.StemDictionary = model.StemDictionary.ValueStringPointer()
	}

	if !model.VecDist.IsUnknown() && !model.VecDist.IsNull() {
		apiField.VecDist = model.VecDist.ValueStringPointer()
	}

	if !model.TokenSeparators.IsUnknown() && !model.TokenSeparators.IsNull() {
		var separators []string
		diags.Append(model.TokenSeparators.ElementsAs(ctx, &separators, false)...)
		apiField.TokenSeparators = &separators
	}

	if !model.SymbolsToIndex.IsUnknown() && !model.SymbolsToIndex.IsNull() {
		var symbols []string
		diags.Append(model.SymbolsToIndex.ElementsAs(ctx, &symbols, false)...)
		apiField.SymbolsToIndex = &symbols
	}

	return diags
}

func (model *CollectionFieldModel) validateCollectionFieldOptions() diag.Diagnostics {
	var diags diag.Diagnostics
	if model.Reference.Equal(types.StringValue("")) {
		diags.AddError("Invalid field reference", "An explicit reference must not be empty. Omit reference when this field has no reference.")
	}

	diags.Append(model.validateCollectionFieldIndexOptions()...)
	diags.Append(model.validateCollectionFieldStemOptions()...)
	diags.Append(model.validateCollectionFieldVectorOptions()...)

	if model.Name.Equal(types.StringValue(".*")) {
		switch {
		case model.Optional.Equal(types.BoolValue(false)):
			diags.AddError("Invalid fallback field option", "Typesense requires the exact .* fallback field to be optional.")
		case model.Facet.Equal(types.BoolValue(true)):
			diags.AddError("Invalid fallback field option", "Typesense does not allow faceting on the exact .* fallback field.")
		case model.Index.Equal(types.BoolValue(false)):
			diags.AddError("Invalid fallback field option", "Typesense requires the exact .* fallback field to be indexed.")
		case !model.Reference.IsUnknown() && !model.Reference.IsNull() && model.Reference.ValueString() != "":
			diags.AddError("Invalid fallback field option", "Typesense does not allow a reference on the exact .* fallback field.")
		}
	}

	if option := model.ignoredFallbackFieldOption(); option != "" {
		diags.AddError("Unsupported fallback field option", fmt.Sprintf("Typesense ignores %s on the fallback field %q.", option, ".*"))
	}

	return diags
}

// Typesense constructs the exact .* fallback using only its basic field options.
// Reject ignored options before creating or altering a collection.
func (model *CollectionFieldModel) ignoredFallbackFieldOption() string {
	if model.Name.IsUnknown() || model.Name.IsNull() || model.Name.ValueString() != ".*" {
		return ""
	}

	switch {
	case !model.NumDim.IsUnknown() && !model.NumDim.IsNull():
		return "num_dim"
	case model.Store.Equal(types.BoolValue(false)):
		return "store"
	case model.RangeIndex.Equal(types.BoolValue(true)):
		return "range_index"
	case model.Stem.Equal(types.BoolValue(true)):
		return "stem"
	case model.StemDictionary.ValueString() != "":
		return "stem_dictionary"
	case !model.VecDist.IsUnknown() && !model.VecDist.IsNull():
		return "vec_dist"
	case !model.TokenSeparators.IsUnknown() && !model.TokenSeparators.IsNull() && len(model.TokenSeparators.Elements()) > 0:
		return "token_separators"
	case !model.SymbolsToIndex.IsUnknown() && !model.SymbolsToIndex.IsNull() && len(model.SymbolsToIndex.Elements()) > 0:
		return "symbols_to_index"
	default:
		return ""
	}
}

func (model *CollectionFieldModel) validateCollectionFieldIndexOptions() diag.Diagnostics {
	var diags diag.Diagnostics

	name, fieldType := model.Name.ValueString(), model.Type.ValueString()
	if !model.Sort.IsUnknown() && !model.Sort.IsNull() && !model.Sort.ValueBool() && collectionFieldIsGeo(fieldType) {
		diags.AddError("Invalid field sort", fmt.Sprintf("Geo field %q requires sort = true for Typesense GeoSearch.", name))
	}

	if !model.RangeIndex.IsUnknown() && !model.RangeIndex.IsNull() && model.RangeIndex.ValueBool() && !collectionFieldIsNumeric(fieldType) {
		diags.AddError("Invalid field range_index", fmt.Sprintf("Field %q must be numerical to use range_index = true.", name))
	}

	return diags
}

func (model *CollectionFieldModel) validateCollectionFieldStemOptions() diag.Diagnostics {
	var diags diag.Diagnostics

	name, fieldType := model.Name.ValueString(), model.Type.ValueString()
	if !model.Stem.IsUnknown() && !model.Stem.IsNull() && model.Stem.ValueBool() && !collectionFieldIsString(fieldType) {
		diags.AddError("Invalid field stem", fmt.Sprintf("Field %q must be string or string[] to enable stemming.", name))
	}

	if !model.StemDictionary.IsUnknown() && !model.StemDictionary.IsNull() && model.StemDictionary.ValueString() != "" &&
		!model.Stem.IsUnknown() && !model.Stem.IsNull() && !model.Stem.ValueBool() {
		diags.AddError("Conflicting stemming settings", fmt.Sprintf("Field %q cannot set stem = false with a nonempty stem_dictionary.", name))
	}

	return diags
}

func (model *CollectionFieldModel) validateCollectionFieldVectorOptions() diag.Diagnostics {
	var diags diag.Diagnostics

	name, fieldType := model.Name.ValueString(), model.Type.ValueString()
	if !model.VecDist.IsUnknown() && !model.VecDist.IsNull() &&
		(fieldType != "float[]" || model.NumDim.IsNull() || model.NumDim.IsUnknown()) {
		diags.AddError("Invalid field vec_dist", fmt.Sprintf("Field %q must be a float[] vector with num_dim to set vec_dist.", name))
	}

	return diags
}
