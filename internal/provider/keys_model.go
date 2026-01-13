package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KeysModel struct {
	Keys types.List `tfsdk:"keys"`
}
