package provider_test

import (
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccGenerateScopedSearchKeyFunction(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				output "typesense_scoped_search_key" {
					value = provider::typesense::generate_scoped_search_key("abcdef", { "scope" : "foo", "something" : "123", "another" : ["1", 2], "yetanother" : { "a" : 1, "b" : [2, "3"] } })
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckOutput("typesense_scoped_search_key", base64.StdEncoding.EncodeToString([]byte("iy27OvahxUoH+TD/dNy4fNYc6S7Zjt+W/v0bKs6ugcE=abcd{\"another\":[\"1\",2],\"scope\":\"foo\",\"something\":\"123\",\"yetanother\":{\"a\":1,\"b\":[2,\"3\"]}}"))),
				),
			},
		},
	})
}
