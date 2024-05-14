package resource_key_test

import (
	"context"
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider/resource_key"
	"github.com/stretchr/testify/assert"
)

func TestKeyResourceSchema(t *testing.T) {
	t.Parallel()

	schema := resource_key.KeyResourceSchema(context.Background())
	assert.NotNil(t, schema)

	assert.EqualValues(t, 0, schema.GetVersion())
}
