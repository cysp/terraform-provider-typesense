package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AliasModel struct {
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
	Name           types.String   `tfsdk:"name"`
	CollectionName types.String   `tfsdk:"collection_name"`
}
