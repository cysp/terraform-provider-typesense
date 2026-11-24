package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
)

func TestGetCollectionForAlterationInvalidResponse(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		status      int
		contentType string
		body        string
		syntaxError bool
	}{
		{"trailing JSON", http.StatusOK, "application/json", "{}{}", true},
		{"malformed not found", http.StatusNotFound, "application/json", "{", true},
		{"unexpected content type", http.StatusOK, "text/plain", "unexpected response", false},
		{"forbidden", http.StatusForbidden, "text/plain", "permission denied", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var requests atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", test.contentType)
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			t.Cleanup(server.Close)

			client, err := newTypesenseAPI(server.URL, "test-key")
			require.NoError(t, err)
			collection, err := client.getCollectionForAlteration(t.Context(), "posts")
			require.Nil(t, collection)
			require.EqualValues(t, 1, requests.Load())

			if test.syntaxError {
				require.ErrorAs(t, err, new(*json.SyntaxError))

				return
			}

			var httpErr *typesense.HTTPError
			require.ErrorAs(t, err, &httpErr)
			require.Equal(t, test.status, httpErr.Status)
			require.Equal(t, test.body, string(httpErr.Body))
		})
	}
}
