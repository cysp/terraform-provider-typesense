package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func (model *AliasModel) ResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: model.ResourceSchemaAttributes(ctx),
	}
}

func (model *AliasModel) ResourceSchemaAttributes(_ context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"collection_name": schema.StringAttribute{
			Required: true,
		},
	}
}
