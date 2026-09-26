package provider_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionDynamicOptionalDefaults(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := fmt.Sprintf(`resource "typesense_collection" "test" {
 name = %q
 fields = [
  {name="title",type="string"},
  {name=".*",type="auto"},
  {name="meta_.*",type="string"},
  {name="score",type="auto"},
  {name="tags",type="string*"},
 ]
}`, name)

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.optional", "false"),
			resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.optional", "true"),
			resource.TestCheckResourceAttr("typesense_collection.test", "fields.2.optional", "true"),
			resource.TestCheckResourceAttr("typesense_collection.test", "fields.3.optional", "true"),
			resource.TestCheckResourceAttr("typesense_collection.test", "fields.4.optional", "true"),
		)},
		{Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
		}}},
	}})
}

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

func TestAccCollectionFallbackReplacementIsDiagnosed(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := func(fieldType string) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
 name = %q
 fields = [{name=".*",type=%q}]
}`, name, fieldType)
	}

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config("auto")},
		{Config: config("string"), ExpectError: regexp.MustCompile("cannot replace an existing .* fallback")},
		{Config: config("auto"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
		}}},
	}})
}

func TestAccCollectionRemoveDisjointDynamicRule(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	config := func(rule string) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
 name = %q
 fields = [{name="title",type="string"}%s]
}`, name, rule)
	}
	original := config(`,{name="meta_.*",type="string"}`)
	remaining := config("")

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: original},
		{Config: remaining, PreConfig: func() {
			_, err := client.Collection(name).Documents().Create(context.Background(), map[string]any{"id": "one", "title": "Original", "meta_note": "Retained"}, &api.DocumentIndexParameters{})
			require.NoError(t, err)
		}},
		{Config: remaining, PreConfig: func() {
			collection, err := client.Collection(name).Retrieve(context.Background())
			require.NoError(t, err)
			require.Len(t, collection.Fields, 1)
			require.Equal(t, "title", collection.Fields[0].Name)

			document, err := client.Collection(name).Document("one").Retrieve(context.Background())
			require.NoError(t, err)
			require.Equal(t, "Retained", document["meta_note"])

			result, err := client.Collection(name).Documents().Search(context.Background(), &api.SearchCollectionParams{Q: new("Original"), QueryBy: new("title")})
			require.NoError(t, err)
			require.Equal(t, new(1), result.Found)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
		}}},
	}})
}
