package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionEmbeddingTypeTransition(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		credentials bool
	}{
		{"local model", false}, {"masked remote model", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			model := map[string]string{"model_name": "model"}
			if test.credentials {
				model["api_key"] = "masked"
			}

			original := map[string]any{"name": "value", "type": "float[]", "num_dim": 3, "vec_dist": "ip", "hnsw_params": map[string]int{"M": 24}, "embed": map[string]any{"from": []string{"title"}, "model_config": model}}
			body, err := json.Marshal(map[string]any{"fields": []map[string]any{original}})
			require.NoError(t, err)
			current, err := decodeObservedCollection(body)
			require.NoError(t, err)

			patch, diags := current.encodeAlteration([]api.Field{{Name: "value", Type: "string"}})
			if test.credentials {
				require.Len(t, diags, 1)
				require.Equal(t, "Cannot reindex embedding field", diags[0].Summary())
				require.Nil(t, patch)

				return
			}

			require.Empty(t, diags)

			want, err := json.Marshal(map[string]any{"fields": []map[string]any{{"name": "value", "type": "string"}}})
			require.NoError(t, err)
			require.JSONEq(t, string(want), string(patch))
		})
	}
}
