package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccCollectionResourceDefaults(t *testing.T) {
	t.Parallel()

	collectionName := "testacc_" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	defaultConfig := fmt.Sprintf(`
	resource "typesense_collection" "test" {
		name = %[1]q

		fields = [
			{
				name = "title"
				type = "string"
			},
		]
	}
	`, collectionName)
	facetConfig := fmt.Sprintf(`
	resource "typesense_collection" "test" {
		name = %[1]q

		fields = [
			{
				name = "title"
				type = "string"
				facet = true
			},
		]
	}
	`, collectionName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: defaultConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.facet", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.index", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.infix", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.locale", ""),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.optional", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.sort", "false"),
				),
			},
			{
				Config: defaultConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
					},
				},
			},
			{
				Config: facetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.facet", "true"),
			},
			{
				Config: defaultConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.facet", "false"),
			},
		},
	})
}
