package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func (model *KeysModel) DataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: model.DataSourceSchemaAttributes(ctx),
	}
}

func (*KeysModel) DataSourceSchemaAttributes(ctx context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"keys": schema.ListNestedAttribute{
			NestedObject: schema.NestedAttributeObject{
				Attributes: (&KeyModel{}).DataSourceSchemaAttributes(ctx, false),
			},
			Computed: true,
		},
	}
}
