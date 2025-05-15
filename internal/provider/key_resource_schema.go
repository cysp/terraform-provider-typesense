package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (model *KeyModel) ResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: model.ResourceSchemaAttributes(ctx),
	}
}

func (model *KeyModel) ResourceSchemaAttributes(_ context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed: true,
		},
		"description": schema.StringAttribute{
			Required: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"actions": schema.ListAttribute{
			ElementType: types.StringType,
			Required:    true,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
		"collections": schema.ListAttribute{
			ElementType: types.StringType,
			Required:    true,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
		"expires_at": schema.Int64Attribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		"value": schema.StringAttribute{
			Optional:  true,
			Computed:  true,
			Sensitive: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"value_prefix": schema.StringAttribute{
			Computed: true,
		},
	}
}
