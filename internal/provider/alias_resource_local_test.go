package provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAliasResourceLocal(t *testing.T) {
	t.Parallel()

	aliases := map[string]typesense_api.CollectionAlias{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleLocalAliasRequest(t, aliases, w, req)
	}))
	t.Cleanup(server.Close)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
				resource "typesense_alias" "test" {
					name = "posts"
					collection_name = "posts_v1"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_alias.test", "name", "posts"),
					resource.TestCheckResourceAttr("typesense_alias.test", "collection_name", "posts_v1"),
				),
			},
			{
				Config: providerConfig(server.URL) + `
				resource "typesense_alias" "test" {
					name = "posts"
					collection_name = "posts_v2"
				}
				`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("typesense_alias.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("typesense_alias.test", "collection_name", "posts_v2"),
			},
		},
	})

	assert.Empty(t, aliases)
}

func handleLocalAliasRequest(t *testing.T, aliases map[string]typesense_api.CollectionAlias, w http.ResponseWriter, req *http.Request) {
	t.Helper()

	aliasName := pathSuffix(req.URL.Path, "/aliases/")
	if aliasName == "" {
		http.NotFound(w, req)

		return
	}

	switch req.Method {
	case http.MethodPut:
		var schema typesense_api.CollectionAliasSchema

		err := json.NewDecoder(req.Body).Decode(&schema)
		if err != nil {
			t.Errorf("decode alias schema: %v", err)
			http.Error(w, "invalid alias schema", http.StatusBadRequest)

			return
		}

		alias := typesense_api.CollectionAlias{
			CollectionName: schema.CollectionName,
			Name:           &aliasName,
		}
		aliases[aliasName] = alias
		writeJSON(t, w, http.StatusOK, alias)
	case http.MethodGet:
		alias, ok := aliases[aliasName]
		if !ok {
			http.NotFound(w, req)

			return
		}

		writeJSON(t, w, http.StatusOK, alias)
	case http.MethodDelete:
		alias, ok := aliases[aliasName]
		if !ok {
			http.NotFound(w, req)

			return
		}

		delete(aliases, aliasName)
		writeJSON(t, w, http.StatusOK, alias)
	default:
		http.Error(w, "unexpected method: "+req.Method, http.StatusMethodNotAllowed)
	}
}
