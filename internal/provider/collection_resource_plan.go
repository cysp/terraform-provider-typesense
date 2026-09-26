package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithModifyPlan = (*collectionResource)(nil)

func (r *collectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var previous, planned types.List
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("fields"), &previous)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("fields"), &planned)...)

	if resp.Diagnostics.HasError() {
		return
	}

	oldFallback, oldFound := collectionExactFallback(previous)
	newFallback, newFound := collectionExactFallback(planned)

	if !oldFound || !newFound || !collectionFieldValueKnown(oldFallback) || !collectionFieldValueKnown(newFallback) {
		return
	}

	if !oldFallback.Equal(newFallback) {
		resp.Diagnostics.AddAttributeError(path.Root("fields"), "Cannot replace Typesense fallback in one update", "The exact .* fallback cannot be dropped and readded in one Typesense schema alteration. Remove it in one apply and add the desired declaration in a second apply, or use a new collection and alias cutover.")
	}
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
