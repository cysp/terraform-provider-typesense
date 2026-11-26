package provider_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionResourceRetainUnstoredIndex(t *testing.T) {
	requireLiveAcceptance(t)
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := api.NewClientWithResponses(os.Getenv("TYPESENSE_URL"), api.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	require.NoError(t, err)
	created, err := client.CreateCollectionWithResponse(t.Context(), api.CollectionSchema{Name: name, Fields: []api.Field{{Name: "sku", Type: "string", Optional: new(true), Store: new(false)}, {Name: "title", Type: "string"}}})
	require.NoError(t, err)
	require.NotNil(t, created.JSON201, string(created.Body))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.DeleteCollectionWithResponse(ctx, name)
		require.NoError(t, err)
	})

	config := func(fields string) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 fields=[{name="title",type="string"}%s]
}`, name, fields)
	}

	const (
		retained = `,{name="sku",type="string",optional=true}`
		faceted  = `,{name="sku",type="string",optional=true,facet=true}`
	)

	checkSearch := func() {
		result, err := client.SearchCollectionWithResponse(t.Context(), name, &api.SearchCollectionParams{Q: new("unicorn"), QueryBy: new("sku")})
		require.NoError(t, err)
		require.NotNil(t, result.JSON200, string(result.Body))
		require.Equal(t, new(1), result.JSON200.Found)
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config(retained), ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: name, ImportStatePersist: true},
		{Config: config(faceted), ExpectError: regexp.MustCompile("cannot reindex field")},
		{Config: config(faceted), PreConfig: func() {
			doc, err := client.IndexDocumentWithResponse(t.Context(), name, nil, map[string]any{"id": "one", "title": "part", "sku": "unicorn"})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, doc.StatusCode())
			checkSearch()
		}, ExpectError: regexp.MustCompile("cannot reindex field")},
		{Config: config(retained + `,{name="extra",type="string",optional=true}`), PreConfig: checkSearch},
		{Config: config(""), PreConfig: checkSearch},
	}})
}

func TestAccCollectionResourceUnstoredReindexSendsNoPatch(t *testing.T) {
	t.Parallel()

	collections := map[string]api.CollectionResponse{"posts": {Name: "posts", Fields: []api.Field{{Name: "sku", Type: "string", Store: new(false)}}}}

	var patches atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patches.Add(1)
		}

		handleLocalCollectionRequest(t, collections, w, r)
	}))
	t.Cleanup(server.Close)
	config := providerConfig(server.URL) + `resource "typesense_collection" "test" {
 name="posts"
 fields=[{name="sku",type="string",facet=true}]
}`
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: "posts", ImportStatePersist: true},
		{Config: config, ExpectError: regexp.MustCompile("cannot reindex field")},
	}})
	require.Zero(t, patches.Load())
}
