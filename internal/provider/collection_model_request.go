package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

	for _, field := range fields {
		apiField, fieldDiags := field.ToAPIField()
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

func (model *CollectionFieldModel) ToAPIField() (typesense_api.Field, diag.Diagnostics) {
	var diags diag.Diagnostics

	if model.NumDim.ValueInt64() > int64(int(^uint(0)>>1)) {
		diags.AddError("Invalid field num_dim", fmt.Sprintf("num_dim %d overflows int", model.NumDim.ValueInt64()))

		return typesense_api.Field{}, diags
	}

	apiField := typesense_api.Field{
		Name: model.Name.ValueString(),
		Type: model.Type.ValueString(),
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

	return apiField, diags
}
