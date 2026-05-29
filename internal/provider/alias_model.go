package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type AliasModel struct {
	Name           types.String `tfsdk:"name"`
	CollectionName types.String `tfsdk:"collection_name"`
}
