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

func TestCollectionReindexRequestRemovesInapplicableProperties(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		observed   map[string]any
		targetType string
	}{
		{"remove reference", map[string]any{"name": "field", "type": "string", "reference": "authors.id", "async_reference": true, "cascade_delete": false}, "string"},
		{"remove vector dimensions", map[string]any{"name": "field", "type": "float[]", "num_dim": 3, "vec_dist": "ip", "hnsw_params": map[string]int{"M": 24}}, "float[]"},
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

			expected, err := json.Marshal(map[string]any{"fields": []map[string]any{{"name": "field", "type": "", "drop": true}, {"name": "field", "type": test.targetType}}})
			require.NoError(t, err)
			require.JSONEq(t, string(expected), string(request))
		})
	}
}
