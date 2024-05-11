package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/datasource_key"
	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var (
	_ datasource.DataSource              = (*keyDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*keyDataSource)(nil)
)

//nolint:ireturn
func NewKeyDataSource() datasource.DataSource {
	return &keyDataSource{}
}

type keyDataSource struct {
	providerData TypesenseProviderData
}

func (d *keyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key"
}

func (d *keyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	util.ProviderDataFromDataSourceConfigureRequest(req, &d.providerData, resp)
}

func (d *keyDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_key.KeyDataSourceSchema(ctx)
}

func (d *keyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_key.KeyModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keyID := data.Id.ValueInt64()

	retrievedAPIKey, err := d.providerData.client.Key(keyID).Retrieve(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving key", err.Error())

		return
	}

	resp.Diagnostics.Append(data.ReadFromResponse(ctx, retrievedAPIKey)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
