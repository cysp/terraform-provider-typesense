package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func (model *CollectionModel) ReadFromResponse(ctx context.Context, collection *typesense_api.CollectionResponse) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Name = types.StringValue(collection.Name)
	model.DefaultSortingField = types.StringPointerValue(collection.DefaultSortingField)
	model.EnableNestedFields = types.BoolPointerValue(collection.EnableNestedFields)
	model.CreatedAt = types.Int64PointerValue(collection.CreatedAt)
	model.NumDocuments = types.Int64PointerValue(collection.NumDocuments)

	model.Fields = util.DiagnosticsAppender(types.ListValueFrom(ctx, CollectionFieldObjectType(), CollectionFieldModelsFromAPI(collection.Fields)))(&diags)

	if collection.SymbolsToIndex != nil {
		model.SymbolsToIndex = util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, *collection.SymbolsToIndex))(&diags)
	} else {
		model.SymbolsToIndex = types.ListNull(types.StringType)
	}

	if collection.TokenSeparators != nil {
		model.TokenSeparators = util.DiagnosticsAppender(types.ListValueFrom(ctx, types.StringType, *collection.TokenSeparators))(&diags)
	} else {
		model.TokenSeparators = types.ListNull(types.StringType)
	}

	return diags
}

func CollectionFieldModelsFromAPI(apiFields []typesense_api.Field) []CollectionFieldModel {
	fields := make([]CollectionFieldModel, 0, len(apiFields))

	for _, apiField := range apiFields {
		field := CollectionFieldModel{
			Name:      types.StringValue(apiField.Name),
			Type:      types.StringValue(apiField.Type),
			Facet:     types.BoolPointerValue(apiField.Facet),
			Index:     types.BoolPointerValue(apiField.Index),
			Infix:     types.BoolPointerValue(apiField.Infix),
			Locale:    types.StringPointerValue(apiField.Locale),
			Optional:  types.BoolPointerValue(apiField.Optional),
			Reference: types.StringPointerValue(apiField.Reference),
			Sort:      types.BoolPointerValue(apiField.Sort),
		}

		if apiField.NumDim != nil {
			field.NumDim = types.Int64Value(int64(*apiField.NumDim))
		} else {
			field.NumDim = types.Int64Null()
		}

		fields = append(fields, field)
	}

	return fields
}
