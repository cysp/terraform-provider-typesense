package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestKeyResourceLocal(t *testing.T) {
	t.Parallel()

	keys := map[int64]typesense_api.ApiKey{}

	var nextID int64 = 1

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleLocalKeyRequest(t, keys, &nextID, w, req)
	}))
	t.Cleanup(server.Close)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
				resource "typesense_key" "test" {
					description = "search posts"
					actions = ["documents:search"]
					collections = ["posts"]
				}

				data "typesense_key" "test" {
					id = typesense_key.test.id
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_key.test", "description", "search posts"),
					resource.TestCheckResourceAttr("typesense_key.test", "actions.0", "documents:search"),
					resource.TestCheckResourceAttr("typesense_key.test", "collections.0", "posts"),
					resource.TestCheckResourceAttr("data.typesense_key.test", "description", "search posts"),
				),
			},
		},
	})

	assert.Empty(t, keys)
}

func handleLocalKeyRequest(
	t *testing.T,
	keys map[int64]typesense_api.ApiKey,
	nextID *int64,
	w http.ResponseWriter,
	req *http.Request,
) {
	t.Helper()

	switch {
	case req.Method == http.MethodPost && req.URL.Path == "/keys":
		var schema typesense_api.ApiKeySchema

		err := json.NewDecoder(req.Body).Decode(&schema)
		if err != nil {
			t.Errorf("decode key schema: %v", err)
			http.Error(w, "invalid key schema", http.StatusBadRequest)

			return
		}

		keyID := *nextID
		*nextID = keyID + 1
		value := fmt.Sprintf("test-key-%d", keyID)
		key := typesense_api.ApiKey{
			Id:          &keyID,
			Description: schema.Description,
			Actions:     schema.Actions,
			Collections: schema.Collections,
			ExpiresAt:   schema.ExpiresAt,
			Value:       &value,
		}
		keys[keyID] = key
		writeJSON(t, w, http.StatusCreated, key)
	case req.Method == http.MethodGet:
		keyID, found := keyIDFromPath(t, req.URL.Path)
		if !found {
			http.NotFound(w, req)

			return
		}

		key, ok := keys[keyID]
		if !ok {
			http.NotFound(w, req)

			return
		}

		key.Value = nil
		writeJSON(t, w, http.StatusOK, key)
	case req.Method == http.MethodDelete:
		keyID, found := keyIDFromPath(t, req.URL.Path)
		if !found {
			http.NotFound(w, req)

			return
		}

		key, ok := keys[keyID]
		if !ok {
			http.NotFound(w, req)

			return
		}

		delete(keys, keyID)
		writeJSON(t, w, http.StatusOK, key)
	default:
		http.Error(w, fmt.Sprintf("unexpected request: %s %s", req.Method, req.URL.Path), http.StatusMethodNotAllowed)
	}
}

func keyIDFromPath(t *testing.T, path string) (int64, bool) {
	t.Helper()

	keyID, err := strconv.ParseInt(pathSuffix(path, "/keys/"), 10, 64)
	if err != nil {
		assert.NoError(t, err)

		return 0, false
	}

	return keyID, true
}
