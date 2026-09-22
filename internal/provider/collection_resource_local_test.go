package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionResourceLocal(t *testing.T) {
	t.Parallel()

	collections := map[string]typesense_api.CollectionResponse{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleLocalCollectionRequest(t, collections, w, req)
	}))
	t.Cleanup(server.Close)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
				resource "typesense_collection" "test" {
					name = "posts"

					fields = [
						{
							name = "title"
							type = "string"
						},
					]
				}
				`,
				ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectIdentity("typesense_collection.test", map[string]knownvalue.Check{"name": knownvalue.StringExact("posts")})},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", "posts"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.name", "title"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.type", "string"),
				),
			},
			{ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: "posts", ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "name"},
			{ResourceName: "typesense_collection.test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity},

			{
				Config: providerConfig(server.URL) + `
				resource "typesense_collection" "test" {
					name = "posts"

					fields = [
						{
							name = "title"
							type = "string"
							facet = true
						},
					]
				}
				`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.facet", "true"),
			},
		},
	})

	assert.Empty(t, collections)
}

func handleLocalCollectionRequest(
	t *testing.T,
	collections map[string]typesense_api.CollectionResponse,
	w http.ResponseWriter,
	req *http.Request,
) {
	t.Helper()

	switch {
	case req.Method == http.MethodPost && req.URL.Path == "/collections":
		var schema typesense_api.CollectionSchema

		err := json.NewDecoder(req.Body).Decode(&schema)
		if err != nil {
			t.Errorf("decode collection schema: %v", err)
			http.Error(w, "invalid collection schema", http.StatusBadRequest)

			return
		}

		collection := typesense_api.CollectionResponse{
			Name:               schema.Name,
			EnableNestedFields: schema.EnableNestedFields,
			Fields:             schema.Fields,
		}
		collections[schema.Name] = collection
		writeJSON(t, w, http.StatusCreated, collection)
	case req.Method == http.MethodPatch:
		name := pathSuffix(req.URL.Path, "/collections/")

		collection, ok := collections[name]
		if !ok {
			http.NotFound(w, req)

			return
		}

		var patch typesense_api.CollectionUpdateSchema

		err := json.NewDecoder(req.Body).Decode(&patch)
		if err != nil {
			t.Error(err)
			http.Error(w, "bad patch", http.StatusBadRequest)

			return
		}

		for _, field := range patch.Fields {
			if field.Drop != nil && *field.Drop {
				collection.Fields = slices.DeleteFunc(collection.Fields, func(old typesense_api.Field) bool { return old.Name == field.Name })
			} else {
				collection.Fields = append(collection.Fields, field)
			}
		}

		collections[name] = collection

		writeJSON(t, w, http.StatusOK, patch)

	case req.Method == http.MethodGet:
		collectionName := pathSuffix(req.URL.Path, "/collections/")

		collection, ok := collections[collectionName]
		if !ok {
			http.NotFound(w, req)

			return
		}

		writeJSON(t, w, http.StatusOK, collection)
	case req.Method == http.MethodDelete:
		collectionName := pathSuffix(req.URL.Path, "/collections/")

		collection, ok := collections[collectionName]
		if !ok {
			http.NotFound(w, req)

			return
		}

		delete(collections, collectionName)
		writeJSON(t, w, http.StatusOK, collection)
	default:
		http.Error(w, fmt.Sprintf("unexpected request: %s %s", req.Method, req.URL.Path), http.StatusMethodNotAllowed)
	}
}
