package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

var _ resource.ResourceWithModifyPlan = (*collectionResource)(nil)

func (r *collectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var previous, planned CollectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)

	if resp.Diagnostics.HasError() {
		return
	}

	oldFallback, oldFound := collectionExactFallback(previous.Fields)
	newFallback, newFound := collectionExactFallback(planned.Fields)

	if !oldFound || !newFound || !collectionFieldValueKnown(oldFallback) || !collectionFieldValueKnown(newFallback) {
		return
	}

	oldField, oldDiags := collectionFallbackAPIField(ctx, oldFallback)

	newField, newDiags := collectionFallbackAPIField(ctx, newFallback)
	if oldDiags.HasError() || newDiags.HasError() {
		// Apply checks the live schema. A plan whose field values cannot be
		// converted should not be rejected here as a fallback replacement.
		return
	}

	// Terraform Core can decide to replace the resource after this call (for
	// example, -replace or replace_triggered_by). Warn here; Update enforces
	// the live-schema restriction only when Core actually requests an update.
	if !sameCollectionField(oldField, newField) {
		resp.Diagnostics.AddAttributeWarning(path.Root("fields"), "Cannot replace Typesense fallback in one schema alteration", "If an in-place update still requires replacing the fallback, apply rejects it before sending an alteration: Typesense cannot drop and readd the exact .* fallback in one request. Remove it in one apply and add the desired declaration in a second apply, or use a new collection and alias cutover. Whole-collection replacement is allowed and deletes the collection's stored documents.")
	}
}

func collectionFallbackAPIField(ctx context.Context, value types.Object) (api.Field, diag.Diagnostics) {
	var model CollectionFieldModel

	diags := value.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return api.Field{}, diags
	}

	field, fieldDiags := model.ToAPIField(ctx)
	diags.Append(fieldDiags...)

	return field, diags
}

func collectionExactFallback(fields types.List) (types.Object, bool) {
	if fields.IsNull() || fields.IsUnknown() {
		return types.Object{}, false
	}

	for _, value := range fields.Elements() {
		if value.IsNull() || value.IsUnknown() {
			continue
		}

		field := value.(types.Object)                     //nolint:forcetypeassert // The schema guarantees object elements.
		name := field.Attributes()["name"].(types.String) //nolint:forcetypeassert // The schema guarantees a string name.

		if !name.IsNull() && !name.IsUnknown() && name.ValueString() == ".*" {
			return field, true
		}
	}

	return types.Object{}, false
}

func collectionFieldValueKnown(field types.Object) bool {
	for _, value := range field.Attributes() {
		if value.IsUnknown() {
			return false
		}
	}

	return true
}
