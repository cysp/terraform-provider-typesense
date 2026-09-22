package provider

import (
	"context"
	"net/http"
	"net/url"
	"os"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/typesense/typesense-go/v3/typesense"
	typesenseapi "github.com/typesense/typesense-go/v3/typesense/api"
)

var (
	_ provider.Provider              = (*TypesenseProvider)(nil)
	_ provider.ProviderWithFunctions = (*TypesenseProvider)(nil)
)

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
	resp.Schema = (&TypesenseModel{}).Schema(ctx)
}

func (p *TypesenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data TypesenseModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.URL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Unknown API URL", "The Typesense URL must be known before the provider can perform operations.")
	}

	if data.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Unknown API key", "The Typesense API key must be known before the provider can perform operations.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var typesenseURL string
	if !data.URL.IsNull() {
		typesenseURL = data.URL.ValueString()
	} else if typesenseURLFromEnv, found := util.TypesenseURLFromEnv(); found {
		typesenseURL = typesenseURLFromEnv
	}

	endpoint, endpointErr := url.Parse(typesenseURL)

	if typesenseURL == "" {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Missing API URL", "Set url, TYPESENSE_URL, or TYPESENSE_HOST with optional TYPESENSE_PROTOCOL and TYPESENSE_PORT.")
	} else if endpointErr != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Invalid API URL", "Use an absolute HTTP or HTTPS URL with a host and without user information, query parameters, or a fragment.")
	}

	var typesenseAPIKey string
	if !data.APIKey.IsNull() {
		typesenseAPIKey = data.APIKey.ValueString()
	} else {
		if typesenseAPIKeyFromEnv, found := os.LookupEnv("TYPESENSE_API_KEY"); found {
			typesenseAPIKey = typesenseAPIKeyFromEnv
		}
	}

	if typesenseAPIKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing API key", "Set api_key or the TYPESENSE_API_KEY environment variable.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Operation contexts own deadlines. A redirect can replay a mutation or
	// forward credentials, so return the original response instead.
	apiClient, err := typesenseapi.NewClientWithResponses(typesenseURL,
		typesenseapi.WithAPIKey(typesenseAPIKey),
		typesenseapi.WithHTTPClient(&http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}}),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Invalid API URL", "The Typesense client could not initialize the configured endpoint.")

		return
	}

	typesenseClient := typesense.NewClient(typesense.WithAPIClient(apiClient))

	dataSourceData := TypesenseProviderData{client: typesenseClient, keys: keyAPI{client: apiClient}, alter: make(chan struct{}, 1)}
	resp.DataSourceData = dataSourceData
	resp.ResourceData = dataSourceData
}

func (p *TypesenseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "typesense"
	resp.Version = p.version
}

func (p *TypesenseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewKeyDataSource,
		NewKeysDataSource,
	}
}

func (p *TypesenseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAliasResource,
		NewCollectionResource,
		NewKeyResource,
	}
}

func (p *TypesenseProvider) Functions(context.Context) []func() function.Function {
	return []func() function.Function{NewGenerateScopedSearchKeyFunction}
}
