package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestKeyMetadataReadPreservesSecret(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(trial *rapid.T) {
		secret := rapid.String().Draw(trial, "prior secret")
		prefix := rapid.String().Draw(trial, "server prefix")
		model := provider.KeyModel{Value: types.StringValue(secret)}
		diags := model.ReadFromResponse(t.Context(), &provider.KeyResponse{ValuePrefix: &prefix})
		require.Empty(trial, diags)
		require.Equal(trial, types.StringValue(secret), model.Value)
		require.Equal(trial, types.StringValue(prefix), model.ValuePrefix)
	})
}
