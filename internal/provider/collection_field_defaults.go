package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func defaultCollectionFieldSort(field api.Field) bool {
	if !slices.Contains([]string{"int32", "int64", "float", "bool", "geopoint", "geopoint[]", "geopolygon"}, field.Type) {
		return false
	}

	return field.NumDim == nil || (field.Facet != nil && *field.Facet)
}

func collectionFieldIsGeo(fieldType string) bool {
	return slices.Contains([]string{"geopoint", "geopoint[]", "geopolygon"}, fieldType)
}

func collectionFieldIsNumeric(fieldType string) bool {
	return slices.Contains([]string{"int32", "int32[]", "int64", "int64[]", "float", "float[]"}, fieldType)
}

func collectionFieldIsString(fieldType string) bool {
	return fieldType == "string" || fieldType == "string[]"
}

type collectionSortPlanModifier struct{}

func (collectionSortPlanModifier) Description(context.Context) string {
	return "Defaults to the Typesense sort setting for the field type"
}

func (m collectionSortPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (collectionSortPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.ConfigValue.IsUnknown() {
		return
	}

	var fieldType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("type"), &fieldType)...)

	if resp.Diagnostics.HasError() || fieldType.IsUnknown() || fieldType.IsNull() {
		return
	}

	if !req.ConfigValue.IsNull() {
		if !req.ConfigValue.ValueBool() && collectionFieldIsGeo(fieldType.ValueString()) {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid field sort", "Geo fields require sort = true for Typesense GeoSearch.")
		}

		return
	}

	field := api.Field{Type: fieldType.ValueString()}
	if !defaultCollectionFieldSort(field) {
		resp.PlanValue = types.BoolValue(false)

		return
	}

	var (
		dimensions types.Int64
		facet      types.Bool
	)

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("num_dim"), &dimensions)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("facet"), &facet)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if dimensions.IsUnknown() {
		if !facet.IsUnknown() && !facet.IsNull() && facet.ValueBool() {
			resp.PlanValue = types.BoolValue(true)
		} else {
			resp.PlanValue = types.BoolUnknown()
		}

		return
	}

	if facet.IsUnknown() && !dimensions.IsNull() {
		resp.PlanValue = types.BoolUnknown()

		return
	}

	resp.PlanValue = types.BoolValue(dimensions.IsNull() || (!facet.IsNull() && facet.ValueBool()))
}

type collectionStemPlanModifier struct{}

func (collectionStemPlanModifier) Description(context.Context) string {
	return "Defaults to true when a stem dictionary is configured, false otherwise"
}

func (m collectionStemPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (collectionStemPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.ConfigValue.IsUnknown() {
		return
	}

	var dictionary, fieldType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("stem_dictionary"), &dictionary)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("type"), &fieldType)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if req.ConfigValue.IsNull() {
		if dictionary.IsUnknown() {
			resp.PlanValue = types.BoolUnknown()

			return
		}

		resp.PlanValue = types.BoolValue(!dictionary.IsNull() && dictionary.ValueString() != "")
	}

	if dictionary.IsUnknown() || fieldType.IsUnknown() || fieldType.IsNull() {
		return
	}

	if !dictionary.IsNull() && dictionary.ValueString() != "" && !resp.PlanValue.ValueBool() {
		resp.Diagnostics.AddAttributeError(req.Path, "Conflicting stemming settings", "A nonempty stem_dictionary requires stem = true.")
	}

	if !resp.PlanValue.IsUnknown() && resp.PlanValue.ValueBool() && !collectionFieldIsString(fieldType.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid field stem", "Stemming is supported only for string and string[] fields.")
	}
}

type collectionRangeIndexPlanModifier struct{}

func (collectionRangeIndexPlanModifier) Description(context.Context) string {
	return "Validates that range indexing is used with a numerical field"
}

func (m collectionRangeIndexPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (collectionRangeIndexPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() || !req.PlanValue.ValueBool() {
		return
	}

	var fieldType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("type"), &fieldType)...)

	if !fieldType.IsUnknown() && !fieldType.IsNull() && !collectionFieldIsNumeric(fieldType.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid field range_index", "Range indexing is supported only for numerical fields.")
	}
}

type collectionVecDistPlanModifier struct{}

func (collectionVecDistPlanModifier) Description(context.Context) string {
	return "Defaults to cosine for vector fields and is unset for other fields"
}

func (m collectionVecDistPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (collectionVecDistPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsUnknown() {
		return
	}

	var (
		dimensions types.Int64
		fieldType  types.String
	)

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("num_dim"), &dimensions)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("type"), &fieldType)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if req.ConfigValue.IsNull() {
		switch {
		case dimensions.IsUnknown():
			resp.PlanValue = types.StringUnknown()
		case dimensions.IsNull():
			resp.PlanValue = types.StringNull()
		default:
			resp.PlanValue = types.StringValue("cosine")
		}
	}

	if !resp.PlanValue.IsNull() && !resp.PlanValue.IsUnknown() && !dimensions.IsUnknown() && !fieldType.IsUnknown() && !fieldType.IsNull() &&
		(dimensions.IsNull() || fieldType.ValueString() != "float[]") {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid field vec_dist", fmt.Sprintf("vec_dist requires a float[] vector field with num_dim; field type is %q.", fieldType.ValueString()))
	}
}
