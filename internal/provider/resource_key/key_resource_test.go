package resource_key_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider/resource_key"
	"github.com/stretchr/testify/assert"
)

func TestKeyResourceSchema(t *testing.T) {
	t.Parallel()

	schema := resource_key.KeyResourceSchema(t.Context())
	assert.NotNil(t, schema)

	assert.EqualValues(t, 0, schema.GetVersion())
}
