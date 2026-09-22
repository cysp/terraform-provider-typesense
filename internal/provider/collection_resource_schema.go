package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
						MarkdownDescription: "Field name or dynamic pattern. Names in configuration must be unique.",
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
						MarkdownDescription: "Whether documents may omit this field. Defaults to false; set true for dynamic fields.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"reference": schema.StringAttribute{
						MarkdownDescription: "Referenced collection and field for joins. Typesense 29.1 cannot add or modify reference fields on an existing collection; use a new collection or upgrade to 30.2 for those changes.",
						Optional:            true,
					},
					"sort": schema.BoolAttribute{
						MarkdownDescription: "Whether this field is sortable. Defaults to false.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
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
			MarkdownDescription: "Symbols included in the index. Omission uses the API value. Explicit changes require collection replacement.",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"token_separators": schema.ListAttribute{
			MarkdownDescription: "Additional token separators. Omission uses the API value. Explicit changes require collection replacement.",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
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
