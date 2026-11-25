package provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionUpdateUncertainCompletion(t *testing.T) {
	t.Parallel()

	for _, failure := range []string{"server_error", "deadline"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()

			var mutations atomic.Int32

			collections := map[string]api.CollectionResponse{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPatch {
					handleLocalCollectionRequest(t, collections, w, req)

					return
				}

				mutations.Add(1)
				// The server commits the mutation before its successful response is lost.
				handleLocalCollectionRequest(t, collections, httptest.NewRecorder(), req)

				if failure == "deadline" {
					<-req.Context().Done()

					return
				}

				http.Error(w, "response lost after commit", http.StatusServiceUnavailable)
			}))
			t.Cleanup(server.Close)
			config := providerConfig(server.URL) + `resource "typesense_collection" "test" {
    name="posts"
    fields=[{name="title",type="string",facet=false}]
    timeouts={update="100ms"}
   }`
			changed := providerConfig(server.URL) + `resource "typesense_collection" "test" {
    name="posts"
    fields=[{name="title",type="string",facet=true}]
    timeouts={update="100ms"}
   }`
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config},
				{Config: changed, ExpectError: regexp.MustCompile("The request was sent once and was not retried")},
				{Config: changed, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			}})
			require.EqualValues(t, 1, mutations.Load())
			require.Empty(t, collections)
		})
	}
}

func TestAccCollectionOverlappingAlterationSendsNoMutation(t *testing.T) {
	t.Parallel()

	var mutations atomic.Int32

	collections := map[string]api.CollectionResponse{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPatch {
			mutations.Add(1)
		}

		handleLocalCollectionRequest(t, collections, w, req)
	}))
	t.Cleanup(server.Close)
	original := providerConfig(server.URL) + `resource "typesense_collection" "test" {
 name="posts"
 enable_nested_fields=true
 fields=[{name="person",type="object",facet=false}, {name="person.title",type="string"}]
 }`
	changed := strings.ReplaceAll(original, "facet=false", "facet=true")
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: original},
		{Config: changed, ExpectError: regexp.MustCompile("No schema alteration was sent")},
		{Config: original, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
	}})
	require.Zero(t, mutations.Load())
	require.Empty(t, collections)
}

func TestAccCollectionResourceCredentialBackedEmbeddingAlteration(t *testing.T) {
	t.Parallel()

	var (
		existing api.CollectionResponse
		patches  atomic.Int32
	)

	require.NoError(t, json.Unmarshal([]byte(`{"name":"posts","fields":[
		{"name":"title","type":"string","facet":false,"index":true,"infix":false,"locale":"","optional":false,"sort":false},
		{"name":"embedding","type":"float[]","num_dim":3,"facet":false,"index":true,"infix":false,"locale":"","optional":true,"sort":false,
		 "embed":{"from":["title"],"model_config":{"model_name":"openai/example","api_key":"abcde*****"}}}
	]}`), &existing))

	collections := map[string]api.CollectionResponse{"posts": existing}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPatch {
			patches.Add(1)
		}

		handleLocalCollectionRequest(t, collections, w, req)
	}))
	t.Cleanup(server.Close)

	config := providerConfig(server.URL) + `resource "typesense_collection" "test" {
  name = "posts"
  fields = [
    {name = "title", type = "string", facet = false},
    {name = "embedding", type = "float[]", num_dim = 3, optional = true},
  ]
}`
	unrelated := strings.Replace(config, "facet = false", "facet = true", 1)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config + `
import {
  to = typesense_collection.test
  id = "posts"
}`},
			{Config: unrelated},
			{Config: strings.Replace(unrelated, "optional = true", "optional = false", 1), ExpectError: regexp.MustCompile("Cannot reindex embedding field")},
			{Config: strings.Replace(unrelated, `{name = "embedding", type = "float[]", num_dim = 3, optional = true},`, "", 1)},
		},
	})
	require.EqualValues(t, 2, patches.Load(), "only the unrelated change and the explicit field removal should send a PATCH")
}
