package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func CollectionFieldObjectType() basetypes.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":             types.StringType,
			"type":             types.StringType,
			"facet":            types.BoolType,
			"index":            types.BoolType,
			"infix":            types.BoolType,
			"locale":           types.StringType,
			"num_dim":          types.Int64Type,
			"optional":         types.BoolType,
			"reference":        types.StringType,
			"sort":             types.BoolType,
			"store":            types.BoolType,
			"range_index":      types.BoolType,
			"stem":             types.BoolType,
			"stem_dictionary":  types.StringType,
			"vec_dist":         types.StringType,
			"token_separators": types.ListType{ElemType: types.StringType},
			"symbols_to_index": types.ListType{ElemType: types.StringType},
		},
	}
}
