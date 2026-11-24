package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func (model *AliasModel) ResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a Typesense collection alias. Import an existing alias before managing it, even if it already points to the desired collection. When the first lookup reports the alias missing, creation checks again over five seconds before sending the upsert. Coordinate concurrent writers because the lookup and upsert are separate requests. Changing the target of a managed alias updates it in place.",
		Attributes:          model.ResourceSchemaAttributes(ctx),
	}
}

func (model *AliasModel) ResourceSchemaAttributes(ctx context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"timeouts": timeouts.AttributesAll(ctx),
		"name": schema.StringAttribute{
			Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			MarkdownDescription: "Alias name. Changing it replaces the alias. Import using this name.",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"collection_name": schema.StringAttribute{
			Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			MarkdownDescription: "Existing collection targeted by the alias. Updates switch the target in place.",
			Required:            true,
		},
	}
}
