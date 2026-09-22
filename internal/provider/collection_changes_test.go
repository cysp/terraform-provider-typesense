package provider //nolint:testpackage // Test private alteration boundaries.

import (
	"testing"

	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionOpaqueDynamicDrop(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"(?!private_).*", "a.b", "tags[0]", ".."} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rule := api.Field{Name: name, Type: "auto"}
			child := api.Field{Name: "title", Type: "string"}
			current := &api.CollectionResponse{Fields: []api.Field{rule, child}}
			changes, err := collectionFieldChanges(current, []api.Field{child})
			require.ErrorContains(t, err, "No schema alteration was sent")
			require.Nil(t, changes)
			changes, err = collectionFieldChanges(current, nil)
			require.NoError(t, err)
			require.Equal(t, []api.Field{{Name: name, Drop: new(true)}, {Name: "title", Drop: new(true)}}, changes)
		})
	}
}

func TestCollectionLiteralDropEffects(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		nested     bool
		parent     api.Field
		dropsChild bool
	}{
		{"catchall", true, api.Field{Name: ".*", Type: "auto"}, false},
		{"nested parent", true, api.Field{Name: "person", Type: "object"}, true},
		{"nesting disabled", false, api.Field{Name: "person", Type: "object"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			child := api.Field{Name: "person.title", Type: "string"}
			current := &api.CollectionResponse{Fields: []api.Field{test.parent, child}, EnableNestedFields: new(test.nested)}
			changes, err := collectionFieldChanges(current, []api.Field{child})
			require.NoError(t, err)

			expected := []api.Field{{Name: test.parent.Name, Drop: new(true)}}
			if test.dropsChild {
				expected = append(expected, api.Field{Name: child.Name, Drop: new(true)}, child)
			}

			require.Equal(t, expected, changes)
		})
	}
}
