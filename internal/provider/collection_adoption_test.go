package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionExpansionAdoption(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name, declaration string
		mutations         int32
	}{
		{"matching", `{name="score",type="int64",optional=true,sort=true}`, 0},
		{"facet", `{name="score",type="int64",optional=true,sort=true,facet=true}`, 1},
		{"type", `{name="score",type="float",optional=true,sort=true}`, 1},
		{"named_auto", `{name="score",type="auto",optional=true}`, 1},
		{"named_string", `{name="score",type="string*",optional=true}`, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
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

			config := func(explicit string) string {
				return providerConfig(server.URL) + fmt.Sprintf(`resource "typesense_collection" "test" {
     name="scores"
     fields=[{name=".*",type="auto",optional=true}, %s]
    }`, explicit)
			}
			original, adopted := config(""), config(scenario.declaration)

			action := plancheck.ResourceActionUpdate
			if scenario.mutations == 0 {
				action = plancheck.ResourceActionNoop
			}

			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: original},
					{Config: adopted, PreConfig: func() {
						current := collections["scores"]
						current.Fields = append(current.Fields, api.Field{Name: "score", Type: "int64", Optional: new(true), Sort: new(true), Store: new(true), Facet: new(false), Infix: new(false), Index: new(true), Locale: new("")})
						collections["scores"] = current
					}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", action)}}},
					{Config: adopted, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
					{Config: adopted, PreConfig: func() {
						require.Equal(t, scenario.mutations, mutations.Load())

						current := collections["scores"]
						require.Len(t, current.Fields, 2)
						require.NotNil(t, current.Fields[1].Store)
						require.True(t, *current.Fields[1].Store)
					}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
					{Config: original},
					{Config: original, PreConfig: func() {
						require.Equal(t, scenario.mutations+1, mutations.Load())
						require.Len(t, collections["scores"].Fields, 1)
					}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
				},
			})
		})
	}
}
