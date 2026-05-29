package provider

import (
	"context"
	"errors"
	"net/http"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/typesense/typesense-go/v3/typesense"
)

var (
	_ resource.Resource                = (*aliasResource)(nil)
	_ resource.ResourceWithConfigure   = (*aliasResource)(nil)
	_ resource.ResourceWithImportState = (*aliasResource)(nil)
)

//nolint:ireturn
func NewAliasResource() resource.Resource {
	return &aliasResource{}
}

type aliasResource struct {
	providerData TypesenseProviderData
}

func (r *aliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alias"
}

func (r *aliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	util.ProviderDataFromResourceConfigureRequest(req, &r.providerData, resp)
}

func (r *aliasResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = (&AliasModel{}).ResourceSchema(ctx)
}

func (r *aliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *aliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AliasModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	aliasSchema := data.ToAPICollectionAliasSchema()

	createdAlias, err := r.providerData.client.Aliases().Upsert(ctx, data.Name.ValueString(), &aliasSchema)
	if err != nil {
		resp.Diagnostics.AddError("Error creating alias", err.Error())

		return
	}

	data.ReadFromResponse(createdAlias)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AliasModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	retrievedAlias, err := r.providerData.client.Alias(data.Name.ValueString()).Retrieve(ctx)
	if err != nil {
		var httpError *typesense.HTTPError
		if errors.As(err, &httpError) {
			if httpError.Status == http.StatusNotFound {
				resp.Diagnostics.AddWarning("Alias not found", "")
				resp.State.RemoveResource(ctx)

				return
			}
		}

		resp.Diagnostics.AddError("Error retrieving alias", err.Error())

		return
	}

	data.ReadFromResponse(retrievedAlias)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AliasModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	aliasSchema := data.ToAPICollectionAliasSchema()

	updatedAlias, err := r.providerData.client.Aliases().Upsert(ctx, data.Name.ValueString(), &aliasSchema)
	if err != nil {
		resp.Diagnostics.AddError("Error updating alias", err.Error())

		return
	}

	data.ReadFromResponse(updatedAlias)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AliasModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deletedAlias, err := r.providerData.client.Alias(data.Name.ValueString()).Delete(ctx)
	if err != nil {
		var httpError *typesense.HTTPError
		if errors.As(err, &httpError) {
			if httpError.Status == http.StatusNotFound {
				resp.Diagnostics.AddWarning("Alias not found", "")

				return
			}
		}

		resp.Diagnostics.AddError("Error deleting alias", err.Error())

		return
	}

	_ = deletedAlias
}
