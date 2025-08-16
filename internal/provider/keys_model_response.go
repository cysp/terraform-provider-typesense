//nolint:stylecheck
package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	typesense_api "github.com/typesense/typesense-go/typesense/api"
)

func (model *KeysModel) ReadFromResponse(ctx context.Context, apiKeys []*typesense_api.ApiKey) diag.Diagnostics {
	var diags diag.Diagnostics

	modelKeys := make([]KeyModel, 0, len(apiKeys))

	for _, apiKey := range apiKeys {
		if apiKey == nil {
			continue
		}

		keyModel := KeyModel{}
		diags.Append(keyModel.ReadFromResponse(ctx, apiKey)...)

		modelKeys = append(modelKeys, keyModel)
	}

	model.Keys = util.DiagnosticsAppender(types.ListValueFrom(ctx, model.Keys.ElementType(ctx), modelKeys))(&diags)

	return diags
}
