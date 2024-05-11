//nolint:revive,stylecheck
package datasource_keys

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	typesense_api "github.com/typesense/typesense-go/typesense/api"
)

type typesenseApiKey struct {
	Actions     []string `tfsdk:"actions"`
	Collections []string `tfsdk:"collections"`
	Description string   `tfsdk:"description"`
	Id          *int64   `tfsdk:"id"`
	Value       *string  `tfsdk:"value"`
	ValuePrefix *string  `tfsdk:"value_prefix"`
}

func (model *KeysModel) ReadFromResponse(ctx context.Context, apiKeys []*typesense_api.ApiKey) diag.Diagnostics {
	var diags diag.Diagnostics

	typesenseApiKeys := make([]typesenseApiKey, 0, len(apiKeys))

	for _, apiKey := range apiKeys {
		if apiKey == nil {
			continue
		}

		typesenseApiKeys = append(typesenseApiKeys, typesenseApiKey{
			Id:          apiKey.Id,
			Actions:     apiKey.Actions,
			Collections: apiKey.Collections,
			Description: apiKey.Description,
			Value:       apiKey.Value,
			ValuePrefix: apiKey.ValuePrefix,
		})
	}

	model.Keys = util.DiagnosticsAppender(types.ListValueFrom(ctx, model.Keys.ElementType(ctx), typesenseApiKeys))(&diags)

	return diags
}
