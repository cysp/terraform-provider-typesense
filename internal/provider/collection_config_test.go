package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCollectionConfigValidation(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, fields, message string }{
		{"reserved id", `[{name="id",type="string"}]`, "Reserved field name"},
		{"duplicates", `[{name="title",type="string"},{name="title",type="int64"}]`, "Duplicate field declaration"},
		{"two dynamic declarations", `[{name="score",type="auto"},{name="score",type="string*"}]`, "Duplicate field declaration"},
		{"three same-name declarations", `[{name="score",type="auto"},{name="score",type="int64"},{name="score",type="float"}]`, "Duplicate field declaration"},
		{"same-name regex declarations", `[{name="score_.*",type="auto"},{name="score_.*",type="int64"}]`, "Duplicate field declaration"},
		{"empty reference", `[{name="title",type="string",reference=""}]`, "at least 1"},
		{"dimensions", `[{name="vector",type="float[]",num_dim=-1}]`, "must be at least 1"},
		{"geo sort", `[{name="location",type="geopoint",sort=false}]`, "Geo fields other than the exact"},
		{"array sort", `[{name="tags",type="string[]",sort=true}]`, "sort = true is supported only"},
		{"auto sort", `[{name="score",type="auto",sort=true}]`, "sort = true is supported only"},
		{"required auto", `[{name="score",type="auto",optional=false}]`, "non-nested dynamic fields"},
		{"required regex", `[{name="meta_.*",type="string",optional=false}]`, "non-nested dynamic fields"},
		{"two-byte separator", `[{name="title",type="string",token_separators=["ab"]}]`, "length"},
		{"multibyte symbol", `[{name="title",type="string",symbols_to_index=["é"]}]`, "length"},
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

func TestCollectionNestedDynamicOptionalPlan(t *testing.T) {
	t.Parallel()

	for _, fields := range []string{
		`[{name="person.score",type="auto",optional=false}]`,
		`[{name="meta_.*",type="object",optional=false}]`,
	} {
		t.Run(fields, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: providerConfig("http://127.0.0.1:1") + `resource "typesense_collection" "test" {
 name="posts"
 enable_nested_fields=true
 fields=` + fields + `
 }`, PlanOnly: true, ExpectNonEmptyPlan: true},
			}})
		})
	}
}

func TestCollectionGeoFallbackCanDisableSort(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "typesense_collection" "test" {
 name="posts"
 fields=[{name=".*",type="geopoint",optional=true,sort=false}]
 }`, PlanOnly: true, ExpectNonEmptyPlan: true},
	}})
}

func TestCollectionTokenListConfigValidation(t *testing.T) {
	t.Parallel()

	for _, option := range []string{
		`token_separators=["ab"]`,
		`symbols_to_index=["é"]`,
	} {
		t.Run(option, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: providerConfig("http://127.0.0.1:1") + `resource "typesense_collection" "test" {
 name="posts"
 ` + option + `
 fields=[{name="title",type="string"}]
 }`, ExpectError: regexp.MustCompile("length")},
			}})
		})
	}
}

func TestCollectionResolvedIDFieldRejected(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "terraform_data" "field" {
 input="id"
 }
 resource "typesense_collection" "test" {
 name="posts"
 fields=[{name=terraform_data.field.output,type="string"}]
 }`, ExpectError: regexp.MustCompile("Reserved field name")},
	}})
}

func TestCollectionResolvedFieldConstraints(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, input, collection, fields, message string
	}{
		{"dynamic optional", `"score"`, "", `[{name=terraform_data.value.output,type="auto",optional=false}]`, "Invalid dynamic field optional"},
		{"sort type", `"string[]"`, "", `[{name="tags",type=terraform_data.value.output,sort=true}]`, "Invalid field sort"},
		{"field separator", `["ab"]`, "", `[{name="title",type="string",token_separators=terraform_data.value.output}]`, "Invalid Attribute Value Length"},
		{"collection symbol", `["é"]`, `symbols_to_index=terraform_data.value.output`, `[{name="title",type="string"}]`, "Invalid Attribute Value Length"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: providerConfig("http://127.0.0.1:1") + `resource "terraform_data" "value" {
 input=` + test.input + `
 }
 resource "typesense_collection" "test" {
 name="posts"
 ` + test.collection + `
 fields=` + test.fields + `
 }`, ExpectError: regexp.MustCompile(test.message)},
			}})
		})
	}
}

func TestCollectionSameNamePairPlan(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "typesense_collection" "test" {
 name="posts"
 fields=[{name="score",type="auto"},{name="score",type="int64",sort=true}]
 }`, PlanOnly: true, ExpectNonEmptyPlan: true},
	}})
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
 }`, ExpectError: regexp.MustCompile("Unsupported fallback field option")},
	}})
}

func TestCollectionFallbackResolvedNameAndOptionRejectsIgnoredOption(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "terraform_data" "field" {
 input=".*"
 }
 resource "terraform_data" "flag" {
 input=true
 }
 resource "typesense_collection" "test" {
 name="posts"
 fields=[{name=terraform_data.field.output,type="auto",store=terraform_data.flag.output}]
 }`, ExpectError: regexp.MustCompile("Unsupported fallback field option")},
	}})
}

func TestCollectionFallbackComputedFieldsRejected(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: providerConfig("http://127.0.0.1:1") + `resource "terraform_data" "schema" {
 input=[{name=".*",type="auto",store=true}]
 }
 resource "typesense_collection" "test" {
 name="posts"
 fields=terraform_data.schema.output
 }`, ExpectError: regexp.MustCompile("Unsupported fallback field option|Cannot validate computed fallback fields")},
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
