package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccUpgradeFromReleasedProvider(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 fields=[{name="title",type="string"}]
 }
 resource "typesense_alias" "test" {
 name=%q
 collection_name=typesense_collection.test.name
 }
 resource "typesense_key" "test" {
 description="upgrade test"
 actions=["documents:search"]
 collections=[typesense_collection.test.name]
 }
 data "typesense_key" "test" { id=typesense_key.test.id }
 `, name, name+"_alias")
	resource.Test(t, resource.TestCase{Steps: []resource.TestStep{
		{Config: config, ExternalProviders: map[string]resource.ExternalProvider{"typesense": {Source: "cysp/typesense", VersionConstraint: "= 0.0.6"}}},
		{Config: config, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
			plancheck.ExpectResourceAction("typesense_alias.test", plancheck.ResourceActionNoop),
			plancheck.ExpectResourceAction("typesense_key.test", plancheck.ResourceActionNoop),
		}}, Check: resource.TestCheckResourceAttrSet("typesense_key.test", "value")},
	}})
}

func TestAccUpgradeOmittedNumericSortFromReleasedProvider(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	config := func(explicitSort bool) string {
		sortSetting := ""
		if explicitSort {
			sortSetting = ", sort = false"
		}

		return fmt.Sprintf(`resource "typesense_collection" "test" {
  name = %q
  fields = [
    { name = "score", type = "int64", optional = true%s },
    { name = "title", type = "string", optional = true },
  ]
}`, name, sortSetting)
	}
	checkScoreSort := func(expected bool) {
		collection, err := client.Collection(name).Retrieve(t.Context())
		require.NoError(t, err)

		for _, field := range collection.Fields {
			if field.Name == "score" {
				require.NotNil(t, field.Sort)
				require.Equal(t, expected, *field.Sort)

				return
			}
		}

		t.Fatal("score field missing from Typesense")
	}

	resource.Test(t, resource.TestCase{Steps: []resource.TestStep{
		{Config: config(false), ExternalProviders: map[string]resource.ExternalProvider{"typesense": {Source: "cysp/typesense", VersionConstraint: "= 0.0.6"}}},
		{
			Config: config(true), ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			PreConfig: func() {
				checkScoreSort(false)

				_, err := client.Collection(name).Documents().Create(t.Context(), map[string]any{"id": "one", "score": 3, "title": "Original"}, &api.DocumentIndexParameters{})
				require.NoError(t, err)
			},
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}},
		},
		{
			Config: config(false), ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}},
		},
		{
			Config: config(false), ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			PreConfig: func() {
				checkScoreSort(true)

				results, err := client.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("Original"), QueryBy: new("title")})
				require.NoError(t, err)
				require.Equal(t, 1, *results.Found)
			},
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}},
		},
	}})
}
