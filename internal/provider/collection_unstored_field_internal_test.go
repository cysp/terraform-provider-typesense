package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionUnstoredFieldReindex(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		description  string
		source       api.Field
		child        string
		nesting      bool
		removeSource bool
		refuse       bool
	}{
		{"object array ancestor", api.Field{Name: "people", Type: "object[]", Store: new(false)}, "people.contact.title", true, false, true},
		{"dotted object ancestor", api.Field{Name: "person.contact", Type: "object", Store: new(false)}, "person.contact.title", true, false, true},
		{"separate paths", api.Field{Name: "person", Type: "object", Store: new(false)}, "personnel.title", true, false, false},
		{"literal dotted name", api.Field{Name: "person", Type: "string", Store: new(false)}, "person.title", false, false, false},
		{"scalar prefix", api.Field{Name: "person", Type: "string", Store: new(false)}, "person.title", true, false, false},
		{"implicit descendant drop", api.Field{Name: "person", Type: "object", Store: new(false)}, "person.title", true, true, true},
	} {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			child := api.Field{Name: test.child, Type: "string", Store: new(true)}
			current := &api.CollectionResponse{Fields: []api.Field{test.source, child}, EnableNestedFields: new(test.nesting)}

			desired := []api.Field{child}
			if !test.removeSource {
				desired[0].Facet = new(true)
				desired = append(desired, test.source)
			}

			changes, err := collectionFieldChanges(current, desired)
			if test.refuse {
				require.ErrorIs(t, err, errReindexUnstoredField)
				require.Nil(t, changes)

				return
			}

			require.NoError(t, err)
			require.Len(t, changes, 2)
		})
	}
}

func TestCollectionUnstoredFieldRepeatedName(t *testing.T) {
	t.Parallel()

	indexed := api.Field{Name: "score", Type: "int64", Store: new(true), Index: new(true)}

	rule := api.Field{Name: "score", Type: "auto", Store: new(false), Index: new(false), Optional: new(true)}
	for _, fields := range [][]api.Field{{indexed, rule}, {rule, indexed}} {
		current := &api.CollectionResponse{Fields: fields}
		changes, err := collectionFieldChanges(current, []api.Field{indexed})
		require.ErrorIs(t, err, errReindexUnstoredField)
		require.Nil(t, changes)
	}
}
