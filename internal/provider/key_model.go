package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KeyModel struct {
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
	ID          types.Int64    `tfsdk:"id"`
	Description types.String   `tfsdk:"description"`
	Actions     types.List     `tfsdk:"actions"`
	Collections types.List     `tfsdk:"collections"`
	Autodelete  types.Bool     `tfsdk:"autodelete"`
	ExpiresAt   types.Int64    `tfsdk:"expires_at"`
	Value       types.String   `tfsdk:"value"`
	ValuePrefix types.String   `tfsdk:"value_prefix"`
}
