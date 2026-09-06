package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var (
	_ datasource.DataSource              = (*keysDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*keysDataSource)(nil)
)

//nolint:ireturn
func NewKeysDataSource() datasource.DataSource {
	return &keysDataSource{}
}

type keysDataSource struct {
	providerData TypesenseProviderData
}

func (d *keysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keys"
}

func (d *keysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	util.ProviderDataFromDataSourceConfigureRequest(req, &d.providerData, resp)
}

func (d *keysDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = (&KeysModel{}).DataSourceSchema(ctx)
}

func (d *keysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KeysModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	retrievedAPIKeys, err := d.providerData.client.Keys().Retrieve(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving keys", err.Error())

		return
	}

	resp.Diagnostics.Append(data.ReadFromResponse(ctx, retrievedAPIKeys)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
