package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	typesenseapi "github.com/typesense/typesense-go/v3/typesense/api"
)

var (
	_ resource.Resource                = (*collectionResource)(nil)
	_ resource.ResourceWithIdentity    = (*collectionResource)(nil)
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
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("name"), path.Root("name"), req, resp)
}

func (r *collectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CollectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(validatePlannedFallbackFieldConfig(ctx, req.Config, data.Fields)...)

	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(warnResolvedFieldTokenOverrides(ctx, req.Config, data)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := operationContext(ctx, data.Timeouts.Create, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()

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

	resp.Diagnostics.Append(data.readCollection(ctx, createdCollection)...)

	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("name"), data.Name)...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *collectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CollectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := operationContext(ctx, data.Timeouts.Read, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()

	if resp.Diagnostics.HasError() {
		return
	}

	retrievedCollection, err := retrieveWithNotFoundConfirmation(ctx, notFoundConfirmationTimeout, r.providerData.client.Collection(data.Name.ValueString()).Retrieve)
	if err != nil {
		if typesenseNotFound(err) {
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Error retrieving collection", err.Error())

		return
	}

	resp.Diagnostics.Append(data.readCollection(ctx, retrievedCollection)...)

	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("name"), data.Name)...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *collectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CollectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(validatePlannedFallbackFieldConfig(ctx, req.Config, data.Fields)...)

	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(warnResolvedFieldTokenOverrides(ctx, req.Config, data)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := operationContext(ctx, data.Timeouts.Update, defaultCollectionUpdateTimeout, &resp.Diagnostics)
	defer cancel()

	if resp.Diagnostics.HasError() {
		return
	}

	var state CollectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Fields.Equal(state.Fields) {
		state.Timeouts = data.Timeouts

		if resp.Identity != nil {
			resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("name"), state.Name)...)
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

		return
	}

	// Typesense permits one schema alteration per cluster. Coordinate resources
	// sharing this provider configuration, while honoring the operation deadline.
	select {
	case r.providerData.alter <- struct{}{}:
		defer func() { <-r.providerData.alter }()
	case <-ctx.Done():
		resp.Diagnostics.AddError("Collection update timeout", "Timed out waiting for another collection alteration in this provider configuration.")

		return
	}

	// Verify the planned starting schema, or recognize an already-completed change.
	current, err := retrieveWithNotFoundConfirmation(ctx, notFoundConfirmationTimeout, r.providerData.client.Collection(data.Name.ValueString()).Retrieve)
	if err != nil {
		resp.Diagnostics.AddError("Error reading collection before update", err.Error())

		return
	}

	want, wantDiags := data.ToAPICollectionSchema(ctx)

	resp.Diagnostics.Append(wantDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	prior, priorDiags := state.ToAPICollectionSchema(ctx)
	resp.Diagnostics.Append(priorDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !sameCollectionFields(current.Fields, prior.Fields) && !sameCollectionFields(current.Fields, want.Fields) {
		resp.Diagnostics.AddError("Collection schema changed since planning", "No schema alteration was sent. The observed managed field schema matches neither the schema recorded when planning nor the planned result. Run a new plan with refresh enabled and review its field changes before applying again.")

		return
	}

	changes, err := collectionFieldChanges(current, want.Fields)
	if err != nil {
		resp.Diagnostics.AddError("Cannot reconcile collection schema", err.Error())

		return
	}

	resp.Diagnostics.Append(validateCollectionAlteration(current, changes)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if len(changes) > 0 {
		_, err = r.providerData.client.Collection(data.Name.ValueString()).Update(ctx, &typesenseapi.CollectionUpdateSchema{Fields: changes})
		if err != nil {
			resp.Diagnostics.AddError("Collection update did not complete successfully", "The request was sent once and was not retried. Its completion may be uncertain. Wait for schema alteration activity to finish, then refresh and plan again before applying. "+err.Error())

			return
		}

		current, err = r.waitForCollectionUpdate(ctx, want)
		if err != nil {
			resp.Diagnostics.AddError("Cannot verify collection update", "The alteration returned successfully, but the collection could not be verified to match the plan. No mutation was retried. Typesense may have inferred fields absent from configuration, or replicas may still be catching up. Refresh and inspect every field in the next plan before applying again. "+err.Error())

			return
		}
	}

	resp.Diagnostics.Append(data.readCollection(ctx, current)...)

	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("name"), data.Name)...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *collectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CollectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := operationContext(ctx, data.Timeouts.Delete, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.providerData.client.Collection(data.Name.ValueString()).Delete(ctx)
	if err != nil && !typesenseNotFound(err) {
		resp.Diagnostics.AddError("Error deleting collection", err.Error())
	}
}

func (r *collectionResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{"name": identityschema.StringAttribute{RequiredForImport: true, Description: "Typesense collection name."}}}
}
