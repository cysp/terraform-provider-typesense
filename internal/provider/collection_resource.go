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
	_ resource.Resource                = (*collectionResource)(nil)
	_ resource.ResourceWithConfigure   = (*collectionResource)(nil)
	_ resource.ResourceWithImportState = (*collectionResource)(nil)
)

//nolint:ireturn
func NewCollectionResource() resource.Resource {
	return &collectionResource{}
}

type collectionResource struct {
	providerData TypesenseProviderData
}

func (r *collectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (r *collectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	util.ProviderDataFromResourceConfigureRequest(req, &r.providerData, resp)
}

func (r *collectionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = (&CollectionModel{}).ResourceSchema(ctx)
}

func (r *collectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *collectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CollectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	collectionSchema := util.DiagnosticsAppender(data.ToAPICollectionSchema(ctx))(&resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	createdCollection, err := r.providerData.client.Collections().Create(ctx, &collectionSchema)
	if err != nil {
		resp.Diagnostics.AddError("Error creating collection", err.Error())

		return
	}

	resp.Diagnostics.Append(data.ReadFromResponse(ctx, createdCollection)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *collectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CollectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	retrievedCollection, err := r.providerData.client.Collection(data.Name.ValueString()).Retrieve(ctx)
	if err != nil {
		var httpError *typesense.HTTPError
		if errors.As(err, &httpError) {
			if httpError.Status == http.StatusNotFound {
				resp.Diagnostics.AddWarning("Collection not found", "")
				resp.State.RemoveResource(ctx)

				return
			}
		}

		resp.Diagnostics.AddError("Error retrieving collection", err.Error())

		return
	}

	resp.Diagnostics.Append(data.ReadFromResponse(ctx, retrievedCollection)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *collectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CollectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError("Cannot update collection", "")
}

func (r *collectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CollectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deletedCollection, err := r.providerData.client.Collection(data.Name.ValueString()).Delete(ctx)
	if err != nil {
		var httpError *typesense.HTTPError
		if errors.As(err, &httpError) {
			if httpError.Status == http.StatusNotFound {
				resp.Diagnostics.AddWarning("Collection not found", "")

				return
			}
		}

		resp.Diagnostics.AddError("Error deleting collection", err.Error())

		return
	}

	_ = deletedCollection
}
