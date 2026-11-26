package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithModifyPlan = (*collectionResource)(nil)

func (r *collectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var data CollectionModel
	resp.Diagnostics.Append(data.readPlan(ctx, req.Plan)...)
}

// Terraform applies ignore_changes before planning. Check the planned field names
// and repeat the check at apply when previously unknown names have resolved.
func (model *CollectionModel) readPlan(ctx context.Context, plan tfsdk.Plan) diag.Diagnostics {
	diags := plan.Get(ctx, model)
	if diags.HasError() || model.DefaultSortingField.ValueString() == "" || model.Fields.IsUnknown() || model.Fields.IsNull() {
		return diags
	}

	for _, element := range model.Fields.Elements() {
		if element.IsUnknown() || element.IsNull() {
			return diags
		}

		name := element.(types.Object).Attributes()["name"] //nolint:forcetypeassert // The schema defines object elements with a string name.
		if name.IsUnknown() || name.Equal(model.DefaultSortingField) {
			return diags
		}
	}

	diags.AddAttributeError(path.Root("default_sorting_field"), "Missing default sorting field", fmt.Sprintf("The plan retains default_sorting_field %q, but fields does not contain that name. Keep the field or allow the plan to clear the default sorting field. Check any lifecycle.ignore_changes settings for these attributes.", model.DefaultSortingField.ValueString()))

	return diags
}
