package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type TypesenseModel struct {
	APIKey types.String `tfsdk:"api_key"`
	URL    types.String `tfsdk:"url"`
}

func (m *TypesenseModel) Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: m.SchemaAttributes(ctx),
	}
}

func (m *TypesenseModel) SchemaAttributes(_ context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"url": schema.StringAttribute{
			Optional:            true,
			Description:         "Typesense API URL. Alternatively, can be configured using the `TYPESENSE_URL` environment variable. Alternatively alternatively, can be configured using the `TYPESENSE_PROTOCOL`, `TYPESENSE_HOST` and `TYPESENSE_PORT` environment variables.",
			MarkdownDescription: "Typesense API URL. Alternatively, can be configured using the `TYPESENSE_URL` environment variable. Alternatively alternatively, can be configured using the `TYPESENSE_PROTOCOL`, `TYPESENSE_HOST` and `TYPESENSE_PORT` environment variables.",
		},
		"api_key": schema.StringAttribute{
			Optional:            true,
			Description:         "Typesense Admin API Key. Alternatively, can be configured using the `TYPESENSE_API_KEY` environment variable.",
			MarkdownDescription: "Typesense Admin API Key. Alternatively, can be configured using the `TYPESENSE_API_KEY` environment variable.",
		},
	}
}
