package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestKeyResponseShortSecret(t *testing.T) {
	t.Parallel()

	for secret, prefix := range map[string]string{"": "", "a": "a", "abc": "abc", "abcd": "abcd", "abcde": "abcd"} {
		t.Run(secret, func(t *testing.T) {
			t.Parallel()

			var model provider.KeyModel
			assert.Empty(t, model.ReadFromResponse(t.Context(), &api.ApiKey{Value: &secret}))
			assert.Equal(t, types.StringValue(secret), model.Value)
			assert.Equal(t, types.StringValue(prefix), model.ValuePrefix)
			assert.Empty(t, model.ReadFromResponse(t.Context(), &api.ApiKey{}))
			assert.Equal(t, types.StringValue(secret), model.Value)
		})
	}
}
