package provider

import (
	"context"
	"errors"
	"net/http"

	"github.com/cysp/terraform-provider-typesense/internal/provider/resource_key"
	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/typesense/typesense-go/typesense"
)

var (
	_ resource.Resource                = (*keyResource)(nil)
	_ resource.ResourceWithConfigure   = (*keyResource)(nil)
	_ resource.ResourceWithImportState = (*keyResource)(nil)
)

//nolint:ireturn
func NewKeyResource() resource.Resource {
	return &keyResource{}
}

type keyResource struct {
	providerData TypesenseProviderData
}

func (r *keyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key"
}

func (r *keyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	util.ProviderDataFromResourceConfigureRequest(req, &r.providerData, resp)
}

func (r *keyResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_key.KeyResourceSchema(ctx)
}

func (r *keyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	util.ImportStatePassthroughInt64ID(ctx, path.Root("id"), req, resp)
}

func (r *keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_key.KeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keySchema := util.DiagnosticsAppender(data.ToAPIKeySchema(ctx))(&resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	createdKey, err := r.providerData.client.Keys().Create(ctx, &keySchema)
	if err != nil {
		resp.Diagnostics.AddError("Error creating key", err.Error())

		return
	}

	resp.Diagnostics.Append(data.ReadFromResponse(ctx, createdKey)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_key.KeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keyID := data.Id.ValueInt64()

	retrievedAPIKey, err := r.providerData.client.Key(keyID).Retrieve(ctx)
	if err != nil {
		var httpError *typesense.HTTPError
		if errors.As(err, &httpError) {
			if httpError.Status == http.StatusNotFound {
				resp.Diagnostics.AddWarning("Key not found", "")
				resp.State.RemoveResource(ctx)

				return
			}
		}

		resp.Diagnostics.AddError("Error retrieving key", err.Error())

		return
	}

	resp.Diagnostics.Append(data.ReadFromResponse(ctx, retrievedAPIKey)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_key.KeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError("Cannot update key", "")
}

func (r *keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_key.KeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deletedAPIKey, err := r.providerData.client.Key(data.Id.ValueInt64()).Delete(ctx)
	if err != nil {
		var httpError *typesense.HTTPError
		if errors.As(err, &httpError) {
			if httpError.Status == http.StatusNotFound {
				resp.Diagnostics.AddWarning("Key not found", "")

				return
			}
		}

		resp.Diagnostics.AddError("Error deleting key", err.Error())

		return
	}

	_ = deletedAPIKey
}
