package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KeyDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Actions     types.List   `tfsdk:"actions"`
	Collections types.List   `tfsdk:"collections"`
	ExpiresAt   types.Int64  `tfsdk:"expires_at"`
	Value       types.String `tfsdk:"value"`
	ValuePrefix types.String `tfsdk:"value_prefix"`
}

func KeyDataSourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Computed: true,
			},
			"actions": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"collections": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"expires_at": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"value": schema.StringAttribute{
				Optional:  true,
				Computed:  true,
				Sensitive: true,
			},
			"value_prefix": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}
