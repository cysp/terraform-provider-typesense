package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (model *KeyModel) DataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: model.DataSourceSchemaAttributes(ctx),
	}
}

func (*KeyModel) DataSourceSchemaAttributes(_ context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
	}
}
