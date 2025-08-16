package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/cysp/terraform-provider-typesense/internal/typesense-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
	resp.Schema = (&KeyModel{}).ResourceSchema(ctx)
}

func (r *keyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	util.ImportStatePassthroughInt64ID(ctx, path.Root("id"), req, resp)
}

func (r *keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keySchema := util.DiagnosticsAppender(data.ToAPIKeySchema(ctx))(&resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	createdKey, err := r.providerData.client.CreateKey(ctx, typesense.NewOptApiKeySchema(keySchema))
	if err != nil {
		resp.Diagnostics.AddError("Error creating key", err.Error())

		return
	}

	switch createdKey := createdKey.(type) {
	case *typesense.ApiKey:
		resp.Diagnostics.Append(data.ReadFromResponse(ctx, createdKey)...)

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	case *typesense.CreateKeyBadRequest:
		resp.Diagnostics.AddError("Error creating key", "Bad request: "+createdKey.Message)

	case *typesense.CreateKeyConflict:
		resp.Diagnostics.AddError("Error creating key", "Conflict: "+createdKey.Message)

	default:
		resp.Diagnostics.AddError("Error creating key", "")
	}
}

func (r *keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := typesense.GetKeyParams{
		KeyId: data.ID.ValueInt64(),
	}

	retrievedAPIKey, err := r.providerData.client.GetKey(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving key", err.Error())

		return
	}

	switch retrievedAPIKey := retrievedAPIKey.(type) {
	case *typesense.ApiKey:
		resp.Diagnostics.Append(data.ReadFromResponse(ctx, retrievedAPIKey)...)

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	case *typesense.ApiResponse:
		resp.Diagnostics.AddError("Error retrieving key", retrievedAPIKey.Message)

		// var httpError *typesense.HTTPError
		// if errors.As(err, &httpError) {
		// 	if httpError.Status == http.StatusNotFound {
		// 		resp.Diagnostics.AddWarning("Key not found", "")
		// 		resp.State.RemoveResource(ctx)

		// 		return
		// 	}
		// }

	default:
		resp.Diagnostics.AddError("Error retrieving key", "")
	}
}

func (r *keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError("Cannot update key", "")
}

func (r *keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := typesense.DeleteKeyParams{
		KeyId: data.ID.ValueInt64(),
	}

	deletedAPIKey, err := r.providerData.client.DeleteKey(ctx, params)
	if err != nil {
		// var httpError *typesense.HTTPError
		// if errors.As(err, &httpError) {
		// 	if httpError.Status == http.StatusNotFound {
		// 		resp.Diagnostics.AddWarning("Key not found", "")

		// 		return
		// 	}
		// }

		resp.Diagnostics.AddError("Error deleting key", err.Error())

		return
	}

	_ = deletedAPIKey
}
