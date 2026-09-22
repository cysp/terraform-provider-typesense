package provider

import (
	"cmp"
	"context"
	"slices"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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

type KeysModel struct {
	Keys []KeyDataSourceModel `tfsdk:"keys"`
}

func (d *keysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keys"
}

func (d *keysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	util.ProviderDataFromDataSourceConfigureRequest(req, &d.providerData, resp)
}

func (d *keysDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := (&KeyDataSourceModel{}).DataSourceSchemaAttributes(ctx)
	attributes["id"] = schema.Int64Attribute{Computed: true, MarkdownDescription: "Identifier of the key returned by Typesense."}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists metadata for all Typesense API keys. Requires the `keys:list` action. The bootstrap key and scoped search keys are not included. Key secrets are never returned.",
		Attributes: map[string]schema.Attribute{
			"keys": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "API key metadata returned by Typesense, ordered by key id. An empty result is an empty list.",
				NestedObject:        schema.NestedAttributeObject{Attributes: attributes},
			},
		},
	}
}

func (d *keysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, defaultOperationTimeout)
	defer cancel()

	keys, err := d.providerData.client.Keys().Retrieve(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving keys", err.Error())

		return
	}

	data := KeysModel{Keys: make([]KeyDataSourceModel, 0, len(keys))}
	for _, key := range keys {
		if key == nil {
			continue
		}

		var model KeyModel
		resp.Diagnostics.Append(model.ReadFromResponse(ctx, key)...)
		data.Keys = append(data.Keys, KeyDataSourceModel{ID: model.ID, Description: model.Description, Actions: model.Actions, Collections: model.Collections, ExpiresAt: model.ExpiresAt, ValuePrefix: model.ValuePrefix})
	}

	if resp.Diagnostics.HasError() {
		return
	}

	slices.SortFunc(data.Keys, func(a, b KeyDataSourceModel) int { return cmp.Compare(a.ID.ValueInt64(), b.ID.ValueInt64()) })
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
