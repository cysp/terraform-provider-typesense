package datasource_key_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider/datasource_key"
	"github.com/stretchr/testify/assert"
)

func TestKeyDataSourceSchema(t *testing.T) {
	t.Parallel()

	schema := datasource_key.KeyDataSourceSchema(t.Context())
	assert.NotNil(t, schema)

	assert.EqualValues(t, 0, schema.GetVersion())
}
