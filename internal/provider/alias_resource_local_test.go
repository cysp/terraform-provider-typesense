package provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccAliasResourceCreateUpdateDelete(t *testing.T) {
	t.Parallel()

	aliases := map[string]typesense_api.CollectionAlias{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleLocalAliasRequest(t, aliases, w, req)
	}))
	t.Cleanup(server.Close)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
				resource "typesense_alias" "test" {
					name = "posts"
					collection_name = "posts_v1"
				}
				`,
				ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectIdentity("typesense_alias.test", map[string]knownvalue.Check{"name": knownvalue.StringExact("posts")})},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_alias.test", "name", "posts"),
					resource.TestCheckResourceAttr("typesense_alias.test", "collection_name", "posts_v1"),
				),
			},
			{ResourceName: "typesense_alias.test", ImportState: true, ImportStateId: "posts", ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "name"},
			{ResourceName: "typesense_alias.test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity},

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

func TestAccAliasResourceCreateChecksOwnership(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		target   string
		statuses []int
		want     string
	}{
		{"same_target", "posts_v1", []int{http.StatusOK}, "Alias already exists"},
		{"different_target", "posts_v2", []int{http.StatusOK}, "Alias already exists"},
		{"temporarily_missing", "posts_v2", []int{http.StatusNotFound, http.StatusOK}, "Alias already exists"},
		{"forbidden", "", []int{http.StatusForbidden}, "Error checking alias before creation"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var reads, writes atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					writes.Add(1)
					http.Error(w, "unexpected write", http.StatusMethodNotAllowed)

					return
				}

				assert.Equal(t, "/aliases/posts", req.URL.Path)

				index := int(reads.Add(1)) - 1

				status := test.statuses[min(index, len(test.statuses)-1)]
				if status != http.StatusOK {
					http.Error(w, http.StatusText(status), status)

					return
				}

				writeJSON(t, w, http.StatusOK, typesense_api.CollectionAlias{Name: new("posts"), CollectionName: test.target})
			}))
			t.Cleanup(server.Close)

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{Config: providerConfig(server.URL) + `resource "typesense_alias" "test" {
  name            = "posts"
  collection_name = "posts_v1"
}`, ExpectError: regexp.MustCompile(test.want)}},
			})
			assert.EqualValues(t, len(test.statuses), reads.Load())
			assert.Zero(t, writes.Load())
		})
	}
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
