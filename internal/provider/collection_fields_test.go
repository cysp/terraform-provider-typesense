package provider //nolint:testpackage // Test private reconciliation helpers.

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionFieldOrdering(t *testing.T) {
	t.Parallel()

	wildcard := api.Field{Name: ".*", Type: "auto", Optional: new(true)}
	typed := api.Field{Name: "score_.*", Type: "int64", Optional: new(true)}
	auto := api.Field{Name: "age", Type: "auto", Optional: new(true)}

	object := api.Field{Name: "person_.*", Type: "object", Optional: new(true)}
	for _, test := range []struct {
		name            string
		observed, prior []api.Field
		expected        []api.Field
	}{
		{"preserve configured order", []api.Field{{Name: "a", Type: "string"}, {Name: "b", Type: "string"}}, []api.Field{{Name: "b", Type: "string"}, {Name: "a", Type: "string"}}, []api.Field{{Name: "b", Type: "string"}, {Name: "a", Type: "string"}}},
		{"same-name auto expansion", []api.Field{{Name: "age", Type: "int64", Sort: new(true), Optional: new(true)}, auto}, []api.Field{auto}, []api.Field{auto, {Name: "age", Type: "int64", Sort: new(true), Optional: new(true)}}},
		{"typed numeric expansion", []api.Field{typed, {Name: "score_a", Type: "int64", Sort: new(true), Optional: new(true)}}, []api.Field{typed}, []api.Field{typed, {Name: "score_a", Type: "int64", Sort: new(true), Optional: new(true)}}},
		{"object pattern expansion", []api.Field{object, {Name: "person_a", Type: "object", Optional: new(true)}, {Name: "person_a.age", Type: "int64", Sort: new(true), Optional: new(true)}}, []api.Field{object}, []api.Field{object, {Name: "person_a", Type: "object", Optional: new(true)}, {Name: "person_a.age", Type: "int64", Sort: new(true), Optional: new(true)}}},
		{"contradictory facet remains drift", []api.Field{wildcard, {Name: "extra", Type: "string", Facet: new(true), Optional: new(true)}}, []api.Field{wildcard}, []api.Field{wildcard, {Name: "extra", Type: "string", Facet: new(true), Optional: new(true)}}},
		{"external declaration remains drift", []api.Field{wildcard, {Name: "rogue.*", Type: "string", Optional: new(true)}}, []api.Field{wildcard}, []api.Field{wildcard, {Name: "rogue.*", Type: "string", Optional: new(true)}}},
		{"explicit overlap remains authoritative", []api.Field{wildcard, {Name: "title", Type: "string", Optional: new(true)}}, []api.Field{wildcard, {Name: "title", Type: "string", Optional: new(true)}}, []api.Field{wildcard, {Name: "title", Type: "string", Optional: new(true)}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := collectionFields(test.observed, test.prior)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestCollectionFallbackReplacementRequiresSeparateAlterations(t *testing.T) {
	t.Parallel()

	current := &api.CollectionResponse{Fields: []api.Field{{Name: ".*", Type: "auto", Optional: new(true)}}}
	changes, err := collectionFieldChanges(current, []api.Field{{Name: ".*", Type: "string", Optional: new(true)}})
	require.ErrorContains(t, err, "cannot replace an existing .* fallback")
	require.Empty(t, changes)
}

func TestCollectionDynamicRuleDropDisjointPrefix(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, pattern, retained string
		blocked                 bool
	}{
		{"disjoint literal prefix", "meta_.*", "title", false},
		{"matching literal prefix", "meta_.*", "meta_title", true},
		{"regex alternation", "(meta_|title).*", "title", true},
		{"named auto regex", "a.b", "title", true},
		{"catchall", ".*", "title", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fieldType := "string"
			if test.name == "named auto regex" || test.name == "catchall" {
				fieldType = "auto"
			}

			current := &api.CollectionResponse{Fields: []api.Field{{Name: test.pattern, Type: fieldType, Optional: new(true)}, {Name: test.retained, Type: "string"}}}
			changes, err := collectionFieldChanges(current, []api.Field{{Name: test.retained, Type: "string"}})

			if test.blocked {
				require.ErrorIs(t, err, errCollectionDynamicDrop)
				require.Empty(t, changes)
			} else {
				require.NoError(t, err)
				require.Equal(t, []api.Field{{Name: test.pattern, Drop: new(true)}}, changes)
			}
		})
	}
}

func TestFieldTokenListDropsCollectionValues(t *testing.T) {
	t.Parallel()

	list := func(values ...string) types.List {
		elements := make([]attr.Value, len(values))
		for i, value := range values {
			elements[i] = types.StringValue(value)
		}

		return types.ListValueMust(types.StringType, elements)
	}

	require.False(t, fieldTokenListDropsCollectionValues(list("-"), list("-")))
	require.False(t, fieldTokenListDropsCollectionValues(list("-"), list("-", "+")))
	require.True(t, fieldTokenListDropsCollectionValues(list("-", "+"), list("-")))
}

func TestCollectionAlterationBoundary(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name             string
		nested           bool
		current, changes []api.Field
		rejected         bool
	}{
		{"reindex object with inferred child", true, []api.Field{{Name: "person", Type: "object"}, {Name: "person.age", Type: "int64"}}, []api.Field{{Name: "person", Drop: new(true)}, {Name: "person", Type: "object", Facet: new(true)}}, true},
		{"add parent over explicit child", true, []api.Field{{Name: "person.age", Type: "int64"}}, []api.Field{{Name: "person", Type: "object"}}, true},
		{"dotted scalar parent", true, []api.Field{{Name: "a.b.c", Type: "string"}}, []api.Field{{Name: "a.b", Type: "string"}}, true},
		{"dotted scalar with nesting disabled", false, []api.Field{{Name: "a.b.c", Type: "string"}}, []api.Field{{Name: "a.b", Type: "string"}}, false},
		{"new parent without descendants", true, []api.Field{{Name: "unrelated", Type: "string"}}, []api.Field{{Name: "person", Type: "object"}}, false},
		{"child edit under unchanged parent", true, []api.Field{{Name: "person", Type: "object"}, {Name: "person.age", Type: "int64"}}, []api.Field{{Name: "person.age", Drop: new(true)}, {Name: "person.age", Type: "int64", Facet: new(true)}}, false},
		{"remove parent retaining explicit child", true, []api.Field{{Name: "person", Type: "object"}, {Name: "person.age", Type: "int64"}}, []api.Field{{Name: "person", Drop: new(true)}, {Name: "person.age", Drop: new(true)}, {Name: "person.age", Type: "int64"}}, false},
		{"undeveloped dynamic rule is not concrete", true, []api.Field{{Name: "person.age", Type: "auto"}, {Name: "person.extra.*", Type: "string"}}, []api.Field{{Name: "person", Type: "object"}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			diags := validateCollectionAlteration(&api.CollectionResponse{Fields: test.current, EnableNestedFields: new(test.nested)}, test.changes)
			if test.rejected {
				require.True(t, diags.HasError())
				require.Contains(t, diags[0].Detail(), "while nested field")
			} else {
				require.Empty(t, diags)
			}
		})
	}
}
