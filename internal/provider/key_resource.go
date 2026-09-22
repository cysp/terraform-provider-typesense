package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*keyResource)(nil)
	_ resource.ResourceWithIdentity    = (*keyResource)(nil)
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
	if req.Identity != nil && req.ID == "" {
		var id types.Int64
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &id)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)

		return
	}

	util.ImportStatePassthroughInt64ID(ctx, path.Root("id"), req, resp)
}

//nolint:dupl // Keep the framework lifecycle and resource-specific API calls explicit.
func (r *keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := operationContext(ctx, data.Timeouts.Create, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()

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

	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := operationContext(ctx, data.Timeouts.Read, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()

	if resp.Diagnostics.HasError() {
		return
	}

	keyID := data.ID.ValueInt64()

	retrievedAPIKey, err := retrieveWithNotFoundConfirmation(ctx, notFoundConfirmationTimeout, r.providerData.client.Key(keyID).Retrieve)
	if err != nil {
		if typesenseNotFound(err) {
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Error retrieving key", err.Error())

		return
	}

	resp.Diagnostics.Append(data.ReadFromResponse(ctx, retrievedAPIKey)...)

	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := operationContext(ctx, data.Timeouts.Update, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()

	if resp.Diagnostics.HasError() {
		return
	}

	// Remote key attributes require replacement; only local timeouts can update.
	var state KeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = data.Timeouts

	data = state
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), data.ID)...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := operationContext(ctx, data.Timeouts.Delete, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.providerData.client.Key(data.ID.ValueInt64()).Delete(ctx)
	if err != nil && !typesenseNotFound(err) {
		resp.Diagnostics.AddError("Error deleting key", err.Error())
	}
}

func (r *keyResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{"id": identityschema.Int64Attribute{RequiredForImport: true, Description: "Typesense key id."}}}
}
