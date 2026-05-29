package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func (model *AliasModel) ReadFromResponse(alias *typesense_api.CollectionAlias) {
	model.Name = types.StringPointerValue(alias.Name)
	model.CollectionName = types.StringValue(alias.CollectionName)
}
