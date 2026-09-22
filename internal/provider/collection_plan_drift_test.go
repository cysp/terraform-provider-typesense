package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionSchemaDriftAfterPlan(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name     string
		mutate   func(*api.CollectionResponse)
		rejected bool
	}{
		{"added_field", func(current *api.CollectionResponse) {
			field := current.Fields[0]
			field.Name = "remote"
			current.Fields = append(current.Fields, field)
		}, true},
		{"retained_field_changed", func(current *api.CollectionResponse) {
			current.Fields[1].Facet = new(true)
		}, true},
		{"already_desired", func(current *api.CollectionResponse) {
			current.Fields[0].Facet = new(true)
		}, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			var mutex sync.Mutex

			mutations := 0
			collections := map[string]api.CollectionResponse{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mutex.Lock()
				defer mutex.Unlock()

				if req.Method == http.MethodPatch {
					mutations++
				}

				handleLocalCollectionRequest(t, collections, w, req)
			}))
			t.Cleanup(server.Close)

			name := "posts_" + scenario.name
			config := func(facet bool) string {
				return providerConfig(server.URL) + fmt.Sprintf(`resource "typesense_collection" "test" {
  name=%q
  fields=[{name="title",type="string",facet=%t},{name="body",type="string"}]
}`, name, facet)
			}

			var expectedError *regexp.Regexp

			freshAction := plancheck.ResourceActionNoop
			wantMutations := 0

			if scenario.rejected {
				expectedError = regexp.MustCompile("Collection schema changed since planning")
				freshAction = plancheck.ResourceActionUpdate
				wantMutations = 1
			}

			var (
				externalSchema api.CollectionResponse
				plannedFields  []api.Field
			)

			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config(false)},
				{Config: config(true), ExpectError: expectedError, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate),
					collectionPlanCheckFunc(func() {
						mutex.Lock()
						defer mutex.Unlock()

						current := collections[name]
						plannedFields = slices.Clone(current.Fields)
						plannedFields[0].Facet = new(true)

						scenario.mutate(&current)
						current.NumDocuments = new(int64(7))
						collections[name] = current
						externalSchema = current
						externalSchema.Fields = slices.Clone(current.Fields)
					}),
				}}},
				{
					Config: config(true), PreConfig: func() {
						mutex.Lock()
						defer mutex.Unlock()

						require.Zero(t, mutations)
						require.Equal(t, externalSchema, collections[name])
					}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", freshAction)}},
					ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectIdentity("typesense_collection.test", map[string]knownvalue.Check{"name": knownvalue.StringExact(name)})},
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.facet", "true"),
						resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.facet", "false"),
						resource.TestCheckResourceAttr("typesense_collection.test", "num_documents", "7"),
					),
				},
				{Config: config(true), PreConfig: func() {
					mutex.Lock()
					defer mutex.Unlock()

					require.Equal(t, wantMutations, mutations)
					require.ElementsMatch(t, plannedFields, collections[name].Fields)
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			}})

			mutex.Lock()
			defer mutex.Unlock()

			require.Empty(t, collections)
		})
	}
}
