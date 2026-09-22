package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
