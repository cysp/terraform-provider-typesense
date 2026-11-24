package provider //nolint:testpackage // Test private alteration boundaries.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
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

func TestCollectionAlterationSourceProperties(t *testing.T) {
	t.Parallel()

	auto := map[string]any{"name": "phrase", "type": "auto", "store": true, "truncate_len": 91, "future_property": true}

	concrete := map[string]any{"name": "phrase", "type": "string", "store": false, "truncate_len": 7}
	for _, test := range []struct {
		name   string
		fields []map[string]any
		target string
	}{
		{"exact after dynamic", []map[string]any{auto, concrete}, "string"},
		{"exact before dynamic", []map[string]any{concrete, auto}, "string"},
		{"unique name on type change", []map[string]any{concrete}, "string[]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			body, err := json.Marshal(map[string]any{"fields": test.fields})
			require.NoError(t, err)

			observed, err := decodeObservedCollection(body)
			require.NoError(t, err)

			patch, diags := observed.encodeAlteration([]api.Field{{Name: "phrase", Type: test.target, Facet: new(true)}})
			require.Empty(t, diags)

			expected, err := json.Marshal(map[string]any{"fields": []map[string]any{{"name": "phrase", "type": test.target, "facet": true, "store": false, "truncate_len": 7}}})
			require.NoError(t, err)
			require.JSONEq(t, string(expected), string(patch))
		})
	}
}
