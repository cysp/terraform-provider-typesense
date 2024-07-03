package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
					description = ""
					expires_at = 64723363199
				}
				`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("typesense_key.test", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_key.test", "expires_at", "64723363199"),
				),
			},
			{
				Config: `
				resource "typesense_key" "test" {
					actions = ["search:*"]
					collections = ["*"]
					description = ""
					expires_at = 0
				}
				`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("typesense_key.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_key.test", "expires_at", "0"),
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

func TestAccKeyResourceDeleted(t *testing.T) {
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
			},
			{
				Config: `
				resource "typesense_key" "test" {
					actions = ["search:*"]
					collections = ["*"]
					description = ""
				}

				import {
					id = typesense_key.test.id
					to = typesense_key.test_dup
				}

				resource "typesense_key" "test_dup" {
					actions = ["search:*"]
					collections = ["*"]
					description = ""
				}
				`,
			},
			{
				Config: `
				resource "typesense_key" "test" {
					actions = ["search:*"]
					collections = ["*"]
					description = ""
				}
				`,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("typesense_key.test", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}
