package provider

import (
	"context"
	"os"

	"github.com/cysp/terraform-provider-typesense/internal/provider/provider_typesense"
	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/typesense/typesense-go/typesense"
)

var _ provider.Provider = (*TypesenseProvider)(nil)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TypesenseProvider{
			version: version,
		}
	}
}

type TypesenseProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

func (p *TypesenseProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = provider_typesense.TypesenseProviderSchema(ctx)
}

func (p *TypesenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data provider_typesense.TypesenseModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var typesenseURL string
	if !data.Url.IsNull() {
		typesenseURL = data.Url.ValueString()
	} else if typesenseURLFromEnv, found := util.TypesenseURLFromEnv(); found {
		typesenseURL = typesenseURLFromEnv
	}

	if typesenseURL == "" {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Failed to configure client", "No API URL provided")
	}

	var typesenseAPIKey string
	if !data.ApiKey.IsNull() {
		typesenseAPIKey = data.ApiKey.ValueString()
	} else {
		if typesenseAPIKeyFromEnv, found := os.LookupEnv("TYPESENSE_API_KEY"); found {
			typesenseAPIKey = typesenseAPIKeyFromEnv
		}
	}

	if typesenseAPIKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Failed to configure client", "No API key provided")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	typesenseClient := typesense.NewClient(typesense.WithServer(typesenseURL), typesense.WithAPIKey(typesenseAPIKey))

	resp.DataSourceData = TypesenseProviderData{client: typesenseClient}
	resp.ResourceData = TypesenseProviderData{client: typesenseClient}
}

func (p *TypesenseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "typesense"
	resp.Version = p.version
}

func (p *TypesenseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *TypesenseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}
