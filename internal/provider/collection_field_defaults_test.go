package provider_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionFieldServerDefaults(t *testing.T) {
	t.Parallel()

	collections := map[string]api.CollectionResponse{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleLocalCollectionRequest(t, collections, w, req)
	}))
	t.Cleanup(server.Close)

	config := providerConfig(server.URL) + `
resource "typesense_collection" "test" {
  name = "defaults"
  fields = [
    { name = "title", type = "string" },
    { name = "score", type = "int64" },
    { name = "enabled", type = "bool" },
    { name = "location", type = "geopoint" },
    { name = "embedding", type = "float[]", num_dim = 3 },
  ]
}`

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.sort", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.store", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.range_index", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.stem", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.stem_dictionary", ""),
					resource.TestCheckNoResourceAttr("typesense_collection.test", "fields.0.vec_dist"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.token_separators.#", "0"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.symbols_to_index.#", "0"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.2.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.3.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.4.sort", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.4.vec_dist", "cosine"),
				),
			},
			{Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
			}}},
			{ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: "defaults", ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "name"},
		},
	})
}

func TestCollectionFieldRemovalResetsDefaults(t *testing.T) {
	t.Parallel()

	collections := map[string]api.CollectionResponse{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleLocalCollectionRequest(t, collections, w, req)
	}))
	t.Cleanup(server.Close)

	config := func(explicit bool) string {
		score, title, embedding := "", "", ""
		if explicit {
			score = `sort = false, range_index = true, store = false`
			title = `stem_dictionary = "custom", token_separators = ["-"], symbols_to_index = ["+"]`
			embedding = `vec_dist = "ip"`
		}

		return providerConfig(server.URL) + `
resource "typesense_collection" "test" {
  name = "removal"
  fields = [
    { name = "score", type = "int64", ` + score + ` },
    { name = "title", type = "string", ` + title + ` },
    { name = "embedding", type = "float[]", num_dim = 3, ` + embedding + ` },
  ]
}`
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(true), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.sort", "false"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.range_index", "true"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.store", "false"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.stem", "true"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.stem_dictionary", "custom"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.token_separators.0", "-"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.symbols_to_index.0", "+"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.2.vec_dist", "ip"),
			)},
			{ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: "removal", ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "name"},
			{Config: config(false), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate),
			}}, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.sort", "true"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.range_index", "false"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.store", "true"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.stem", "false"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.stem_dictionary", ""),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.token_separators.#", "0"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.symbols_to_index.#", "0"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.2.vec_dist", "cosine"),
			)},
			{Config: config(false), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
			}}},
		},
	})
}
