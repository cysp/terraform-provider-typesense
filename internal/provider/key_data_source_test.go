package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKeyDataSource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "typesense_key" "test" {
					description = ""
					actions = ["search:*"]
					collections = ["*"]
				}

				data "typesense_key" "test" {
					id = typesense_key.test.id
				}

				output "typesense_key_test_action" {
					value = data.typesense_key.test.actions[0]
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckOutput("typesense_key_test_action", "search:*"),
				),
			},
		},
	})
}
