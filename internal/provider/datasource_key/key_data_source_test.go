package datasource_key_test

import (
	"context"
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider/datasource_key"
	"github.com/stretchr/testify/assert"
)

func TestKeyDataSourceSchema(t *testing.T) {
	t.Parallel()

	schema := datasource_key.KeyDataSourceSchema(context.Background())
	assert.NotNil(t, schema)

	assert.EqualValues(t, 0, schema.GetVersion())
}
