package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (model *KeyDataSourceModel) DataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Retrieves metadata for a Typesense API key by its id. Requires the `keys:get` action. This lookup does not return the key secret.",
		Attributes:          model.DataSourceSchemaAttributes(ctx),
	}
}

func (*KeyDataSourceModel) DataSourceSchemaAttributes(_ context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			MarkdownDescription: "Identifier of the existing key to retrieve. Missing keys produce an error.",
			Required:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description returned by Typesense.",
			Computed:            true,
		},
		"actions": schema.ListAttribute{
			MarkdownDescription: "Allowed actions returned by Typesense.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"collections": schema.ListAttribute{
			MarkdownDescription: "Allowed collection names or patterns returned by Typesense.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"expires_at": schema.Int64Attribute{
			MarkdownDescription: "Key expiration as Unix seconds, returned by Typesense.",
			Computed:            true,
		},
		"autodelete": schema.BoolAttribute{
			MarkdownDescription: "Whether Typesense automatically deletes this key after expiration during periodic cleanup.",
			Computed:            true,
		},
		"value_prefix": schema.StringAttribute{
			MarkdownDescription: "Prefix returned by the key metadata endpoint. This is not the complete key secret.",
			Computed:            true,
		},
	}
}
