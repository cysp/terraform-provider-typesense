package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestKeyDataSourceRejectsLegacyInputs(t *testing.T) {
	t.Parallel()

	for _, argument := range []string{`value = "not-a-retrievable-secret"`, `expires_at = 123`} {
		t.Run(argument, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      `data "typesense_key" "test" {` + "\n id = 1\n" + argument + "\n}",
					ExpectError: regexp.MustCompile("Unsupported argument|Invalid Configuration for Read-Only Attribute"),
				}},
			})
		})
	}
}
