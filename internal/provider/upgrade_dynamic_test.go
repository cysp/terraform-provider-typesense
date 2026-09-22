package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccUpgradeDynamicFieldsFromReleasedProvider(t *testing.T) {
	t.Parallel()

	for _, refreshed := range []bool{false, true} {
		t.Run(fmt.Sprintf("old_state_refreshed_%t", refreshed), func(t *testing.T) {
			t.Parallel()

			name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
			config := func(facet, retainAge bool) string {
				age := ""
				if retainAge {
					age = `,{name="person.age",type="int64",optional=true,sort=true}`
				}

				return fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 enable_nested_fields=true
 fields=[
  {name="title",type="string",facet=%t},
  {name=".*",type="auto",optional=true},
  {name="person",type="object",optional=true},
  {name="person.name",type="string",optional=true}%s
 ]
}`, name, facet, age)
			}
			document := map[string]any{"id": "one", "title": "Original", "score": float64(42), "person": map[string]any{"name": "Ada", "age": float64(30)}}
			writeDocument := func() {
				_, err := client.Collection(name).Documents().Create(t.Context(), document, &api.DocumentIndexParameters{})
				require.NoError(t, err)
			}
			oldProvider := map[string]resource.ExternalProvider{"typesense": {Source: "cysp/typesense", VersionConstraint: "= 0.0.6"}}
			steps := []resource.TestStep{{Config: config(true, false), ExternalProviders: oldProvider}}

			beforeUpgrade := writeDocument
			if refreshed {
				steps = append(steps, resource.TestStep{ExternalProviders: oldProvider, PreConfig: writeDocument, RefreshState: true, ExpectNonEmptyPlan: true})
				beforeUpgrade = nil
			}

			steps = append(steps,
				resource.TestStep{
					Config: config(true, false), PreConfig: beforeUpgrade, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}},
					ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectIdentity("typesense_collection.test", map[string]knownvalue.Check{"name": knownvalue.StringExact(name)})},
				},
				resource.TestStep{
					Config: config(false, true), ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					PreConfig: func() {
						observed, err := client.Collection(name).Document("one").Retrieve(t.Context())
						require.NoError(t, err)
						require.Equal(t, document, observed)
						current, err := client.Collection(name).Retrieve(t.Context())
						require.NoError(t, err)

						require.Len(t, current.Fields, 4)
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}},
				},
				resource.TestStep{
					Config: config(false, true), ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					PreConfig: func() {
						observed, err := client.Collection(name).Document("one").Retrieve(t.Context())
						require.NoError(t, err)
						require.Equal(t, document, observed)
						result, err := client.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("*"), FilterBy: new("person.age:=30")})
						require.NoError(t, err)
						require.Equal(t, 1, *result.Found)
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}},
				},
			)
			resource.Test(t, resource.TestCase{Steps: steps})
		})
	}
}
