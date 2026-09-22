package provider_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionUpdateUncertainCompletion(t *testing.T) {
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
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config},
				{Config: changed, ExpectError: regexp.MustCompile("The request was sent once and was not retried")},
				{Config: changed, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			}})
			require.EqualValues(t, 1, mutations.Load())
			require.Empty(t, collections)
		})
	}
}

func TestCollectionOverlappingAlterationSendsNoMutation(t *testing.T) {
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
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: original},
		{Config: changed, ExpectError: regexp.MustCompile("No schema alteration was sent")},
		{Config: original, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
	}})
	require.Zero(t, mutations.Load())
	require.Empty(t, collections)
}
