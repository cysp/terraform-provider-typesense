package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/stretchr/testify/assert"
)

func TestCollectionResourceSchema(t *testing.T) {
	t.Parallel()

	schema := (&provider.CollectionModel{}).ResourceSchema(t.Context())
	assert.NotNil(t, schema)

	assert.EqualValues(t, 0, schema.GetVersion())
}
