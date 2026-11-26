package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionResourceFieldTypeTransition(t *testing.T) {
	requireLiveAcceptance(t)
	t.Parallel()

	client, err := api.NewClientWithResponses(os.Getenv("TYPESENSE_URL"), api.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	require.NoError(t, err)

	for _, test := range []struct {
		name, from, to, option string
		retain                 bool
	}{
		{"remove stemming", "string", "int64", "stem", false},
		{"retain stemming", "string", "string[]", "stem", true},
		{"remove range index", "int64", "string", "range_index", false},
		{"retain range index", "int64", "float[]", "range_index", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			name := createAlterationCollection(t, client, map[string]any{"name": "value", "type": test.from, test.option: true, "sort": false})
			config := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 fields=[{name="value",type=%q}]
}`, name, test.to)
			checkReindexedCollection(t, name, config, func() {
				field := getAlteredField(t, client, name)
				require.JSONEq(t, fmt.Sprintf("%q", test.to), string(field["type"]))

				if test.retain {
					require.Equal(t, "true", string(field[test.option]))

					return
				}

				require.NotEqual(t, "true", string(field[test.option]))
			})
		})
	}
}
