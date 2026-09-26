package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCollectionConfigValidation(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, fields, message string }{
		{"duplicates", `[{name="title",type="string"},{name="title",type="int64"}]`, "Duplicate field declaration"},
		{"dimensions", `[{name="vector",type="float[]",num_dim=-1}]`, "must be at least 1"},
		{"geo sort", `[{name="location",type="geopoint",sort=false}]`, "Geo fields require sort"},
		{"range on string", `[{name="title",type="string",range_index=true}]`, "Range indexing is supported only"},
		{"stem on number", `[{name="score",type="int64",stem=true}]`, "Stemming is supported only"},
		{"dictionary and disabled stem", `[{name="title",type="string",stem_dictionary="custom",stem=false}]`, "nonempty stem_dictionary requires stem"},
		{"distance without vector", `[{name="title",type="string",vec_dist="ip"}]`, "vec_dist requires a float"},
		{"fallback store", `[{name=".*",type="auto",optional=true,store=false}]`, "Unsupported fallback field option"},
		{"fallback explicit default store", `[{name=".*",type="auto",optional=true,store=true}]`, "Unsupported fallback field option"},
		{"fallback separators", `[{name=".*",type="auto",optional=true,token_separators=["-"]}]`, "Unsupported fallback field option"},
		{"fallback empty separators", `[{name=".*",type="auto",optional=true,token_separators=[]}]`, "Unsupported fallback field option"},
		{"fallback required", `[{name=".*",type="auto",optional=false}]`, "Invalid fallback field option"},
		{"fallback facet", `[{name=".*",type="auto",facet=true}]`, "Invalid fallback field option"},
		{"fallback unindexed", `[{name=".*",type="auto",index=false}]`, "Invalid fallback field option"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: providerConfig("http://127.0.0.1:1") + `resource "typesense_collection" "test" {
    name="posts"
    fields=` + test.fields + `
    }`, ExpectError: regexp.MustCompile(test.message)},
			}})
		})
	}
}

func TestCollectionUnknownFieldName(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "terraform_data" "field" {
 input="title"
 }
 resource "typesense_collection" "test" {
 name="posts"
 fields=[{name=terraform_data.field.output,type="string"}]
 }`, PlanOnly: true, ExpectNonEmptyPlan: true},
	}})
}

func TestCollectionFallbackOmittedOptions(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "typesense_collection" "test" {
 name="posts"
 fields=[{name=".*",type="auto",optional=true}]
 }`, PlanOnly: true, ExpectNonEmptyPlan: true},
	}})
}

func TestCollectionFallbackComputedOption(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "terraform_data" "flag" {
 input = true
 }
 resource "typesense_collection" "test" {
 name = "posts"
 fields = [{name=".*",type="auto",store=terraform_data.flag.output}]
 }`, PlanOnly: true, ExpectNonEmptyPlan: true},
	}})
}

func TestCollectionFallbackResolvedNameRejectsIgnoredOptions(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "terraform_data" "field" {
 input=".*"
 }
 resource "typesense_collection" "test" {
 name="posts"
 fields=[{name=terraform_data.field.output,type="auto",optional=true,store=true}]
 }`, ExpectError: regexp.MustCompile("Unsupported fallback field option")},
	}})
}
