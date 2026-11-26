package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionReindexRequest(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		observed   map[string]any
		targetType string
		expected   map[string]any
	}{
		{"remove reference", map[string]any{"name": "field", "type": "string", "reference": "authors.id", "async_reference": true, "cascade_delete": false}, "string", map[string]any{"name": "field", "type": "string"}},
		{"remove vector dimensions", map[string]any{"name": "field", "type": "float[]", "num_dim": 3, "vec_dist": "ip", "hnsw_params": map[string]int{"M": 24}}, "float[]", map[string]any{"name": "field", "type": "float[]"}},
		{"retain stemming dictionary", map[string]any{"name": "field", "type": "string[]", "stem": true, "stem_dictionary": "products"}, "string", map[string]any{"name": "field", "type": "string", "stem": true, "stem_dictionary": "products"}},
		{"remove stemming dictionary", map[string]any{"name": "field", "type": "string[]", "stem": true, "stem_dictionary": "products"}, "int64", map[string]any{"name": "field", "type": "int64"}},
		{"retain range index", map[string]any{"name": "field", "type": "float[]", "range_index": true}, "int32", map[string]any{"name": "field", "type": "int32", "range_index": true}},
		{"remove range index", map[string]any{"name": "field", "type": "int32[]", "range_index": true}, "string", map[string]any{"name": "field", "type": "string"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var (
				request []byte
				readErr error
			)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch || r.URL.Path != "/collections/posts" {
					http.NotFound(w, r)

					return
				}

				request, readErr = io.ReadAll(r.Body)

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{}"))
			}))
			defer server.Close()

			client, err := newTypesenseAPI(server.URL, "test")
			require.NoError(t, err)
			body, err := json.Marshal(map[string]any{"fields": []map[string]any{test.observed}})
			require.NoError(t, err)
			observed, err := decodeObservedCollection(body)
			require.NoError(t, err)
			changes, err := collectionFieldChanges(&observed.schema, []api.Field{{Name: "field", Type: test.targetType}})
			require.NoError(t, err)

			patch, diags := observed.encodeAlteration(changes)
			require.Empty(t, diags)
			err = client.updateCollection(t.Context(), "posts", patch)
			require.NoError(t, err)
			server.Close()
			require.NoError(t, readErr)

			expected, err := json.Marshal(map[string]any{"fields": []map[string]any{{"name": "field", "type": "", "drop": true}, test.expected}})
			require.NoError(t, err)
			require.JSONEq(t, string(expected), string(request))
		})
	}
}
