package provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionResourceReindexStoredProperties(t *testing.T) {
	requireLiveAcceptance(t)
	t.Parallel()

	client, err := api.NewClientWithResponses(os.Getenv("TYPESENSE_URL"), api.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	require.NoError(t, err)

	t.Run("vector", func(t *testing.T) {
		t.Parallel()

		name := createAlterationCollection(t, client, map[string]any{"name": "v", "type": "float[]", "num_dim": 3, "hnsw_params": map[string]int{"M": 24, "ef_construction": 150}})
		config := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 fields=[{name="v",type="float[]",num_dim=3,optional=true}]
}`, name)
		checkReindexedCollection(t, name, config, func() {
			field := getAlteredField(t, client, name)
			require.JSONEq(t, `{"M":24,"ef_construction":150}`, string(field["hnsw_params"]))
			require.Equal(t, "true", string(field["optional"]))
		})
	})
	t.Run("reference", func(t *testing.T) {
		t.Parallel()

		info, err := client.DebugWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, info.JSON200)
		require.NotNil(t, info.JSON200.Version)

		if strings.HasPrefix(*info.JSON200.Version, "29.") {
			t.Skip("Typesense 29.1 rejects alterations to reference fields")
		}

		parent := createAlterationCollection(t, client, map[string]any{"name": "title", "type": "string"})
		name := createAlterationCollection(t, client, map[string]any{"name": "author", "type": "string", "reference": parent + ".id", "async_reference": true, "cascade_delete": false, "truncate_len": 7})
		p, err := client.IndexDocumentWithResponse(t.Context(), parent, nil, map[string]any{"id": "one", "title": "example"})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, p.StatusCode(), string(p.Body))
		child, err := client.IndexDocumentWithResponse(t.Context(), name, nil, map[string]any{"id": "two", "author": "one"})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, child.StatusCode(), string(child.Body))

		config := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 fields=[{name="author",type="string",reference=%q,facet=true}]
}`, name, parent+".id")
		checkReindexedCollection(t, name, config, func() {
			field := getAlteredField(t, client, name)
			require.Equal(t, "true", string(field["facet"]))
			require.Equal(t, "true", string(field["async_reference"]))
			require.Equal(t, "false", string(field["cascade_delete"]))
			require.Equal(t, "7", string(field["truncate_len"]))
			deleted, err := client.DeleteDocumentWithResponse(t.Context(), parent, "one")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, deleted.StatusCode(), string(deleted.Body))
			retained, err := client.GetDocumentWithResponse(t.Context(), name, "two")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, retained.StatusCode(), string(retained.Body))
		})
	})
}

func checkReindexedCollection(t *testing.T, name, config string, check func()) {
	t.Helper()
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: name, ImportStatePersist: true},
		{Config: config},
		{Config: config, PlanOnly: true, PreConfig: check},
	}})
}

func createAlterationCollection(t *testing.T, client *api.ClientWithResponses, field map[string]any) string {
	t.Helper()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	body, err := json.Marshal(map[string]any{"name": name, "fields": []map[string]any{field}})
	require.NoError(t, err)
	created, err := client.CreateCollectionWithBodyWithResponse(t.Context(), "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, created.StatusCode(), string(created.Body))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		deleted, err := client.DeleteCollectionWithResponse(ctx, name)
		require.NoError(t, err)
		require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, deleted.StatusCode())
	})

	return name
}

func getAlteredField(t *testing.T, client *api.ClientWithResponses, name string) map[string]json.RawMessage {
	t.Helper()
	response, err := client.GetCollectionWithResponse(t.Context(), name)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	var observed struct {
		Fields []map[string]json.RawMessage `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(response.Body, &observed))
	require.Len(t, observed.Fields, 1)

	return observed.Fields[0]
}
