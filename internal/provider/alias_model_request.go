package provider

import "github.com/typesense/typesense-go/v3/typesense/api"

func (model *AliasModel) ToAPICollectionAliasSchema() api.CollectionAliasSchema {
	return api.CollectionAliasSchema{
		CollectionName: model.CollectionName.ValueString(),
	}
}
