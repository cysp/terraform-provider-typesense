package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	resp.Schema = (&KeyDataSourceModel{}).DataSourceSchema(ctx)
}

func (d *keyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, defaultOperationTimeout)
	defer cancel()

	keyID := data.ID.ValueInt64()

	retrievedAPIKey, err := retrieveWithNotFoundConfirmation(ctx, notFoundConfirmationTimeout, d.providerData.client.Key(keyID).Retrieve)
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving key", err.Error())

		return
	}

	resp.Diagnostics.Append(data.ReadFromResponse(ctx, retrievedAPIKey)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &KeyDataSourceModel{ID: data.ID, Description: data.Description, Actions: data.Actions, Collections: data.Collections, ExpiresAt: data.ExpiresAt, ValuePrefix: data.ValuePrefix})...)
}

// KeyDataSourceModel describes metadata returned by the key retrieval API.
type KeyDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Actions     types.List   `tfsdk:"actions"`
	Collections types.List   `tfsdk:"collections"`
	ExpiresAt   types.Int64  `tfsdk:"expires_at"`
	ValuePrefix types.String `tfsdk:"value_prefix"`
}
