package provider

import typesense_api "github.com/typesense/typesense-go/v3/typesense/api"

func (model *AliasModel) ToAPICollectionAliasSchema() typesense_api.CollectionAliasSchema {
	return typesense_api.CollectionAliasSchema{
		CollectionName: model.CollectionName.ValueString(),
	}
}
