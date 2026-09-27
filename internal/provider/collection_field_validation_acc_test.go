package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccCollectionFallbackComputedIgnoredOption(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := fmt.Sprintf(`resource "terraform_data" "flag" {
 input = true
}
resource "typesense_collection" "test" {
 name = %q
 fields = [{name=".*",type="auto",store=terraform_data.flag.output}]
}`, name)

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("Unsupported fallback field option")},
	}})
}

func TestAccCollectionComputedEmptyReferenceRejected(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := fmt.Sprintf(`resource "terraform_data" "reference" {
 input = ""
}
resource "typesense_collection" "test" {
 name = %q
 fields = [{name="title",type="string",reference=terraform_data.reference.output}]
}`, name)

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("Invalid field reference|at least 1")},
	}})
}

func TestAccCollectionComputedTokenOverride(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := fmt.Sprintf(`resource "terraform_data" "separators" {
 input = ["-"]
}
resource "typesense_collection" "test" {
 name = %q
 token_separators = ["-", "+"]
 fields = [{name="title",type="string",token_separators=terraform_data.separators.output}]
}`, name)

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config},
		{Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
		}}},
	}})
}
