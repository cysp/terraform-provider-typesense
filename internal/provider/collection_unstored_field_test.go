package provider_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
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

	client, name := createCollectionWithUnstoredField(t, api.CollectionSchema{Fields: []api.Field{{Name: "sku", Type: "string", Optional: new(true), Store: new(false)}, {Name: "title", Type: "string"}}})

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

func createCollectionWithUnstoredField(t *testing.T, schema api.CollectionSchema) (*api.ClientWithResponses, string) {
	t.Helper()

	schema.Name = "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := api.NewClientWithResponses(os.Getenv("TYPESENSE_URL"), api.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	require.NoError(t, err)
	created, err := client.CreateCollectionWithResponse(t.Context(), schema)
	require.NoError(t, err)
	require.NotNil(t, created.JSON201, string(created.Body))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.DeleteCollectionWithResponse(ctx, schema.Name)
		require.NoError(t, err)
	})

	return client, schema.Name
}

func TestAccCollectionResourceRetainUnstoredNestedIndex(t *testing.T) {
	requireLiveAcceptance(t)
	t.Parallel()

	client, name := createCollectionWithUnstoredField(t, api.CollectionSchema{EnableNestedFields: new(true), Fields: []api.Field{
		{Name: "person", Type: "object", Optional: new(true), Store: new(false)},
		{Name: "person.title", Type: "string", Optional: new(true), Store: new(true)},
	}})

	doc, err := client.IndexDocumentWithResponse(t.Context(), name, nil, map[string]any{"id": "one", "person": map[string]string{"title": "unicorn"}})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, doc.StatusCode())

	schema, err := client.GetCollectionWithResponse(t.Context(), name)
	require.NoError(t, err)
	require.NotNil(t, schema.JSON200, string(schema.Body))
	child := slices.IndexFunc(schema.JSON200.Fields, func(field api.Field) bool { return field.Name == "person.title" })
	require.NotEqual(t, -1, child)
	require.Equal(t, new(true), schema.JSON200.Fields[child].Store)

	stored, err := client.GetDocumentWithResponse(t.Context(), name, "one")
	require.NoError(t, err)
	require.NotNil(t, stored.JSON200, string(stored.Body))
	require.NotContains(t, *stored.JSON200, "person")

	checkSearch := func() {
		result, err := client.SearchCollectionWithResponse(t.Context(), name, &api.SearchCollectionParams{Q: new("unicorn"), QueryBy: new("person.title")})
		require.NoError(t, err)
		require.NotNil(t, result.JSON200, string(result.Body))
		require.Equal(t, new(1), result.JSON200.Found)
	}

	config := func(facet bool) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 enable_nested_fields=true
 fields=[{name="person",type="object",optional=true},{name="person.title",type="string",optional=true,facet=%t}]
}`, name, facet)
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config(false), ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: name, ImportStatePersist: true},
		{Config: config(true), PreConfig: checkSearch, ExpectError: regexp.MustCompile(`cannot reindex field "person.title"`)},
		{Config: config(false), PlanOnly: true, PreConfig: checkSearch},
	}})
}

func TestAccCollectionResourceUnstoredReindexSendsNoPatch(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name         string
		fields       []api.Field
		declarations string
	}{
		{"field", []api.Field{{Name: "sku", Type: "string", Store: new(false)}}, `{name="sku",type="string",facet=true}`},
		{"containing object", []api.Field{{Name: "person", Type: "object", Store: new(false), Index: new(true), Optional: new(false), Facet: new(false), Sort: new(false), Infix: new(false), Locale: new("")}, {Name: "person.title", Type: "string", Store: new(true)}}, `{name="person",type="object"},{name="person.title",type="string",facet=true}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			collections := map[string]api.CollectionResponse{"posts": {Name: "posts", Fields: test.fields, EnableNestedFields: new(true)}}

			var patches atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch {
					patches.Add(1)
				}

				handleLocalCollectionRequest(t, collections, w, r)
			}))
			t.Cleanup(server.Close)
			config := providerConfig(server.URL) + fmt.Sprintf(`resource "typesense_collection" "test" {
 name="posts"
 enable_nested_fields=true
 fields=[%s]
}`, test.declarations)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config, ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: "posts", ImportStatePersist: true},
				{Config: config, ExpectError: regexp.MustCompile("cannot reindex field")},
			}})
			require.Zero(t, patches.Load())
		})
	}
}
