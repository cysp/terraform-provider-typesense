package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type CollectionModel struct {
	Name                types.String `tfsdk:"name"`
	Fields              types.List   `tfsdk:"fields"`
	DefaultSortingField types.String `tfsdk:"default_sorting_field"`
	EnableNestedFields  types.Bool   `tfsdk:"enable_nested_fields"`
	SymbolsToIndex      types.List   `tfsdk:"symbols_to_index"`
	TokenSeparators     types.List   `tfsdk:"token_separators"`
	CreatedAt           types.Int64  `tfsdk:"created_at"`
	NumDocuments        types.Int64  `tfsdk:"num_documents"`
}

type CollectionFieldModel struct {
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Facet     types.Bool   `tfsdk:"facet"`
	Index     types.Bool   `tfsdk:"index"`
	Infix     types.Bool   `tfsdk:"infix"`
	Locale    types.String `tfsdk:"locale"`
	NumDim    types.Int64  `tfsdk:"num_dim"`
	Optional  types.Bool   `tfsdk:"optional"`
	Reference types.String `tfsdk:"reference"`
	Sort      types.Bool   `tfsdk:"sort"`
}
