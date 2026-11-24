package provider_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionResourceStemmingRepair(t *testing.T) {
	t.Parallel()
	t.Run("external repair", func(t *testing.T) { t.Parallel(); testStemmingRepair(t, true, plancheck.ResourceActionNoop) })
	t.Run("Terraform repair", func(t *testing.T) { t.Parallel(); testStemmingRepair(t, false, plancheck.ResourceActionUpdate) })
}

func testStemmingRepair(t *testing.T, externalRepair bool, action plancheck.ResourceActionType) {
	t.Helper()

	field := api.Field{Name: "score", Type: "int64", Sort: new(true), Stem: new(true), StemDictionary: new("business")}
	collections := map[string]api.CollectionResponse{"books": {Name: "books", Fields: []api.Field{field}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleLocalCollectionRequest(t, collections, w, req)
	}))
	t.Cleanup(server.Close)
	config := providerConfig(server.URL) + `resource "typesense_collection" "books" {
 name="books"
 fields=[{name="score",type="int64"}]
}`
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{ResourceName: "typesense_collection.books", ImportState: true, ImportStateId: "books", ImportStatePersist: true, Config: config},
		{Config: config, PreConfig: func() {
			if !externalRepair {
				return
			}

			field.Stem = new(false)
			field.StemDictionary = new("")
			collections["books"] = api.CollectionResponse{Name: "books", Fields: []api.Field{field}}
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.books", action)}}},
	}})
}
