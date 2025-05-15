package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KeyModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Actions     types.List   `tfsdk:"actions"`
	Collections types.List   `tfsdk:"collections"`
	ExpiresAt   types.Int64  `tfsdk:"expires_at"`
	Value       types.String `tfsdk:"value"`
	ValuePrefix types.String `tfsdk:"value_prefix"`
}
