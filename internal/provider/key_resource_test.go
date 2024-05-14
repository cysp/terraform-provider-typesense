package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKeyResource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "typesense_key" "test" {
					actions = ["search:*"]
					collections = ["*"]
					description = ""
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_key.test", "id"),
					resource.TestMatchResourceAttr("typesense_key.test", "value", regexp.MustCompile("^.+$")),
					resource.TestMatchResourceAttr("typesense_key.test", "value_prefix", regexp.MustCompile("^.{4}$")),
				),
			},
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_key.test", "id"),
					resource.TestMatchResourceAttr("typesense_key.test", "value", regexp.MustCompile("^.+$")),
					resource.TestMatchResourceAttr("typesense_key.test", "value_prefix", regexp.MustCompile("^.{4}$")),
				),
			},
			{
				Config: `
				resource "typesense_key" "test" {
					actions = ["search:*"]
					collections = ["*"]
					description = "testacc key"
				}

				resource "typesense_key" "test_clone" {
					actions = ["search:*"]
					collections = ["*"]
					description = "testacc key clone"
					value = typesense_key.test.value
				}
				`,
				ExpectError: regexp.MustCompile("Error creating key"),
			},
		},
	})
}

func TestAccKeyResourceImport(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "typesense_key" "test" {
					actions = ["search:*"]
					collections = ["*"]
					description = ""
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_key.test", "id"),
				),
			},
			{
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
				ResourceName:            "typesense_key.test",
			},
		},
	})
}
