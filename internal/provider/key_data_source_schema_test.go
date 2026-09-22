package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/stretchr/testify/assert"
)

func TestKeyDataSourceSchema(t *testing.T) {
	t.Parallel()

	schema := (&provider.KeyDataSourceModel{}).DataSourceSchema(t.Context())
	assert.NotNil(t, schema)

	assert.EqualValues(t, 0, schema.GetVersion())
	assert.True(t, schema.Attributes["id"].IsRequired())
	assert.True(t, schema.Attributes["expires_at"].IsComputed())
	assert.False(t, schema.Attributes["expires_at"].IsOptional())
	assert.NotContains(t, schema.Attributes, "value")
}
