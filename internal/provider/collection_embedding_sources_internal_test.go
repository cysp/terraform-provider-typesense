package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionEmbeddingSourceDrops(t *testing.T) {
	t.Parallel()

	parent := api.Field{Name: "person", Type: "object"}
	title := api.Field{Name: "person.title", Type: "string"}
	body := api.Field{Name: "body", Type: "string"}
	embedding := api.Field{Name: "vec", Type: "float[]", NumDim: new(3)}
	model, err := json.Marshal(map[string]any{"from": []string{body.Name, title.Name}, "model_config": map[string]string{"model_name": "model"}})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(model, &embedding.Embed))
	current := &api.CollectionResponse{EnableNestedFields: new(true), Fields: []api.Field{parent, title, body, embedding}}

	// Dropping the parent implicitly drops its child before re-adding the child.
	_, err = collectionFieldChanges(current, []api.Field{title, body, embedding})
	require.ErrorContains(t, err, `embedding field "vec" depends on "person.title"`)

	// The first source can be removed together with the embedding.
	patch, err := collectionFieldChanges(current, []api.Field{parent, title})
	require.NoError(t, err)
	require.Equal(t, []api.Field{{Name: "body", Drop: new(true)}, {Name: "vec", Drop: new(true)}}, patch)
}
