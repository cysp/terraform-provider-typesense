package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCollectionConfigValidation(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, fields, message string }{
		{"duplicates", `[{name="title",type="string"},{name="title",type="int64"}]`, "Duplicate field declaration"},
		{"dimensions", `[{name="vector",type="float[]",num_dim=-1}]`, "must be at least 1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: providerConfig("http://127.0.0.1:1") + `resource "typesense_collection" "test" {
    name="posts"
    fields=` + test.fields + `
    }`, ExpectError: regexp.MustCompile(test.message)},
			}})
		})
	}
}

func TestAccCollectionUnknownFieldName(t *testing.T) {
	t.Parallel()

	config := providerConfig("http://127.0.0.1:1") + `resource "terraform_data" "field" {
 input="other"
 }
 resource "typesense_collection" "test" {
 name="posts"
 default_sorting_field="rank"
 fields=[{name=terraform_data.field.output,type="int64",sort=true}]
 }`
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
		{Config: config, ExpectError: regexp.MustCompile("Missing default sorting field")},
	}})
}
