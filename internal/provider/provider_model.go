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
		MarkdownDescription: "Manage Typesense collections, aliases, and API keys. Tested with Terraform 1.15/1.16 and Typesense 30.2.",
		Attributes:          m.SchemaAttributes(ctx),
	}
}

func (m *TypesenseModel) SchemaAttributes(_ context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"url": schema.StringAttribute{
			Optional:            true,
			Description:         "Absolute HTTP or HTTPS Typesense endpoint. Resolution order: url, TYPESENSE_URL, then TYPESENSE_HOST with optional TYPESENSE_PROTOCOL and TYPESENSE_PORT. URLs must not contain credentials, query parameters, or fragments.",
			MarkdownDescription: "Absolute HTTP or HTTPS Typesense endpoint. Resolution order: url, TYPESENSE_URL, then TYPESENSE_HOST with optional TYPESENSE_PROTOCOL and TYPESENSE_PORT. URLs must not contain credentials, query parameters, or fragments.",
		},
		"api_key": schema.StringAttribute{
			Optional:            true,
			Sensitive:           true,
			Description:         "Typesense API key with permission for the managed operations. Uses TYPESENSE_API_KEY when omitted.",
			MarkdownDescription: "Typesense API key with permission for the managed operations. Uses TYPESENSE_API_KEY when omitted.",
		},
	}
}
