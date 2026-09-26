package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (model *CollectionModel) ResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a Typesense collection and its complete observed field schema. Supported schema changes update the existing collection in place.",
		Attributes:          model.ResourceSchemaAttributes(ctx),
	}
}

func (model *CollectionModel) ResourceSchemaAttributes(ctx context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"timeouts": timeouts.AttributesAll(ctx),
		"name": schema.StringAttribute{
			Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			MarkdownDescription: "Collection name. Changing the collection name requires replacement.",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"fields": schema.ListNestedAttribute{
			MarkdownDescription: "Complete field schema, including fields inferred by Typesense. Fields returned by the API but absent from configuration are drift; applying removes their indexes while retaining stored document values. Supported changes alter the existing collection and can block writes while reindexing. See the [collection lifecycle guide](../guides/collection-lifecycle) for ownership and recovery.",
			Required:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
						MarkdownDescription: "Field name or dynamic pattern. Do not declare the reserved `id` field; Typesense manages it automatically and omits it from collection schemas. A named `auto` or `string*` declaration may share its name with one concrete field; other duplicate names are invalid.",
						Required:            true,
					},
					"type": schema.StringAttribute{
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
						MarkdownDescription: "Typesense field type, for example string, int64, object, auto, or an array type.",
						Required:            true,
					},
					"facet": schema.BoolAttribute{
						MarkdownDescription: "Whether this field is facetable. Defaults to false.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"index": schema.BoolAttribute{
						MarkdownDescription: "Whether to index this field. Defaults to true.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"infix": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable infix searching. Defaults to false.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"locale": schema.StringAttribute{
						MarkdownDescription: "Locale used for tokenization. Defaults to the empty string.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
					},
					"num_dim": schema.Int64Attribute{
						Validators:          []validator.Int64{int64validator.AtLeast(1)},
						MarkdownDescription: "Positive number of dimensions for a vector field.",
						Optional:            true,
					},
					"optional": schema.BoolAttribute{
						MarkdownDescription: "Whether documents may omit this field. Defaults to true for dynamic fields and false for other fields. Non-nested dynamic fields require true; with nested fields enabled, object/object[] fields and dotted names without .* may set false.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							collectionOptionalPlanModifier{},
						},
					},
					"reference": schema.StringAttribute{
						MarkdownDescription: "Referenced collection and field for joins. Omit for a field without a reference; an explicit empty string is invalid.",
						Optional:            true,
						Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"sort": schema.BoolAttribute{
						MarkdownDescription: "Whether this field is sortable. Defaults to true for scalar int32, int64, float, bool and geo fields, and false for other field types. Only these types and scalar string can set true; geo fields other than the exact `.*` fallback cannot set false. Removing an explicit value resets to this type-specific default and can reindex an existing field.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							collectionSortPlanModifier{},
						},
					},
					"store": schema.BoolAttribute{
						MarkdownDescription: "Whether Typesense stores this field's value in documents. Defaults to true. Setting false omits the value from subsequent stored documents; older stored values are not purged, and restoring true cannot recover omitted values. On Typesense 30.2, fields with store = false have shown search index loss after snapshot and restart; see the collection lifecycle guide.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"range_index": schema.BoolAttribute{
						MarkdownDescription: "Whether to build an index optimized for range filtering on a numerical field. Defaults to false.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						PlanModifiers: []planmodifier.Bool{
							collectionRangeIndexPlanModifier{},
						},
					},
					"stem": schema.BoolAttribute{
						MarkdownDescription: "Whether to stem a string or string[] field. Defaults to false, or true when stem_dictionary is nonempty. An explicit false conflicts with a nonempty dictionary.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							collectionStemPlanModifier{},
						},
					},
					"stem_dictionary": schema.StringAttribute{
						MarkdownDescription: "Name of the stemming dictionary for a string or string[] field. Defaults to the empty string. A nonempty value enables stemming.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
					},
					"vec_dist": schema.StringAttribute{
						MarkdownDescription: "Vector distance metric, cosine or ip. Defaults to cosine when num_dim declares a float[] vector field; unset for other fields. Typesense 29.1 can report cosine after snapshot and restart for a field created with ip; see the collection lifecycle guide.",
						Optional:            true,
						Computed:            true,
						Validators:          []validator.String{stringvalidator.OneOf("cosine", "ip")},
						PlanModifiers: []planmodifier.String{
							collectionVecDistPlanModifier{},
						},
					},
					"token_separators": schema.ListAttribute{
						MarkdownDescription: "Single-byte token separators for this field. A nonempty list overrides the collection-level separators; an empty list inherits them.",
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, nil)),
						Validators:          []validator.List{listvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 1))},
					},
					"symbols_to_index": schema.ListAttribute{
						MarkdownDescription: "Single-byte symbols to index for this field. A nonempty list overrides the collection-level symbols; an empty list inherits them.",
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, nil)),
						Validators:          []validator.List{listvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 1))},
					},
				},
			},
		},
		"default_sorting_field": schema.StringAttribute{
			MarkdownDescription: "Default sorting field returned by Typesense when omitted. Explicit changes require collection replacement.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"enable_nested_fields": schema.BoolAttribute{
			MarkdownDescription: "Whether nested object fields are enabled. Explicit changes require collection replacement.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"symbols_to_index": schema.ListAttribute{
			MarkdownDescription: "Single-byte symbols included in the index. Omission uses the API value. Explicit changes require collection replacement.",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			Validators:          []validator.List{listvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 1))},
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"token_separators": schema.ListAttribute{
			MarkdownDescription: "Additional single-byte token separators. Omission uses the API value. Explicit changes require collection replacement.",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			Validators:          []validator.List{listvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 1))},
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"created_at": schema.Int64Attribute{
			MarkdownDescription: "Collection creation time as Unix seconds.",
			Computed:            true,
		},
		"num_documents": schema.Int64Attribute{
			MarkdownDescription: "Current number of stored documents, refreshed from Typesense.",
			Computed:            true,
		},
	}
}
