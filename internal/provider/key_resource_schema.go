package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (model *KeyModel) ResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a Typesense API key. Changes to permissions, description, expiration or the secret replace the key.",
		Attributes:          model.ResourceSchemaAttributes(ctx),
	}
}

func (model *KeyModel) ResourceSchemaAttributes(ctx context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"timeouts": timeouts.AttributesAll(ctx),
		"id": schema.Int64Attribute{
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			MarkdownDescription: "Typesense key identifier. Import using its decimal representation.",
			Computed:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the key. Changes replace the key.",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"actions": schema.ListAttribute{
			MarkdownDescription: "Allowed Typesense actions. Changes replace the key.",
			ElementType:         types.StringType,
			Required:            true,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
		"collections": schema.ListAttribute{
			MarkdownDescription: "Collection names or patterns permitted by this key. Changes replace the key.",
			ElementType:         types.StringType,
			Required:            true,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
		"expires_at": schema.Int64Attribute{
			MarkdownDescription: "Expiration as Unix seconds. When omitted, Typesense sets a far-future expiration date. Changes replace the key.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"value": schema.StringAttribute{
			MarkdownDescription: "Secret API key. Omit to generate a key. Typesense returns it only on creation; the provider retains it in state across refresh. Import cannot recover it. Changes replace the key. Sensitive values are still stored in Terraform state.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"value_prefix": schema.StringAttribute{
			MarkdownDescription: "Prefix returned by Typesense when retrieving key metadata.",
			Computed:            true,
		},
	}
}
