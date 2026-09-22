package provider_test

import (
	"context"
	"fmt"
	"net/http"
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

func TestAccCollectionExactSchema(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name, fields, field string
		declarations        []string
	}{
		{"lookahead", `{name="(?!private_).*",type="string",optional=true}`, "public_title", []string{"(?!private_).*"}},
		{"backreference", `{name="(tag)_\\1.*",type="string",optional=true}`, "tag_tag_title", []string{`(tag)_\1.*`}},
		{"unicode", `{name="..",type="auto",optional=true}`, "é", []string{".."}},
		{"string_star_with_fallback", `{name=".*",type="auto",optional=true},{name="price[U]SD",type="string*",optional=true,facet=true}`, "priceUSD", []string{".*", "price[U]SD"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
			config := fmt.Sprintf(`resource "typesense_collection" "test" {
    name=%q
    fields=[%s]
   }`, name, scenario.fields)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config},
				{Config: config, PreConfig: func() {
					_, err := client.Collection(name).Documents().Create(t.Context(), map[string]any{"id": "one", scenario.field: "hello"}, &api.DocumentIndexParameters{})
					require.NoError(t, err)
					result, err := client.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("hello"), QueryBy: new(scenario.field)})
					require.NoError(t, err)
					require.Equal(t, new(1), result.Found)
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
				{Config: config, PreConfig: func() {
					current, err := client.Collection(name).Retrieve(t.Context())
					require.NoError(t, err)

					declarations := make([]string, 0, len(current.Fields))
					for _, field := range current.Fields {
						declarations = append(declarations, field.Name)
					}

					require.ElementsMatch(t, scenario.declarations, declarations)
					document, err := client.Collection(name).Document("one").Retrieve(t.Context())
					require.NoError(t, err)
					require.Equal(t, "hello", document[scenario.field])
					_, err = client.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("hello"), QueryBy: new(scenario.field)})

					var response *typesense.HTTPError
					require.ErrorAs(t, err, &response)
					require.Equal(t, http.StatusNotFound, response.Status)
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			}})
		})
	}
}

func TestAccCollectionIgnoreFields(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct{ name, pattern, kind string }{{"regex", "(?!private_).*", "string"}, {"named_auto", "public_title", "auto"}} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
			config := func(facet bool, timeout string) string {
				return fmt.Sprintf(`resource "typesense_collection" "test" {
  name=%q
  fields=[{name=%q,type=%q,optional=true,facet=%t}]
  timeouts={update=%q}
  lifecycle {ignore_changes=[fields]}
 }`, name, scenario.pattern, scenario.kind, facet, timeout)
			}
			checkSearch := func() {
				result, err := client.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("hello"), QueryBy: new("public_title")})
				require.NoError(t, err)
				require.Equal(t, new(1), result.Found)
			}
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config(false, "5m")},
				{Config: config(true, "5m"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
				{Config: config(true, "6m"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate),
					collectionPlanCheckFunc(func() {
						_, err := client.Collection(name).Documents().Create(t.Context(), map[string]any{"id": "one", "public_title": "hello"}, &api.DocumentIndexParameters{})
						require.NoError(t, err)
						checkSearch()
					}),
				}}},
				{Config: config(true, "6m"), PreConfig: checkSearch, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			}})
		})
	}
}

func TestAccCollectionInferredFieldsAfterAlteration(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	config := func(rule string) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
  name=%q
  fields=[{name="title",type="string"},%s]
  timeouts={update="2s"}
 }`, name, rule)
	}
	original := config("")
	changed := config(`{name="meta_.*",type="string",optional=true}`)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: original},
		{Config: changed, PreConfig: func() {
			_, err := client.Collection(name).Documents().Create(t.Context(), map[string]any{"id": "one", "title": "hello", "meta_title": "world"}, &api.DocumentIndexParameters{})
			require.NoError(t, err)
		}, ExpectError: regexp.MustCompile("Cannot verify collection update")},
		{Config: changed, PreConfig: func() {
			current, err := client.Collection(name).Retrieve(t.Context())
			require.NoError(t, err)
			require.Len(t, current.Fields, 3)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
		{Config: changed, PreConfig: func() {
			current, err := client.Collection(name).Retrieve(t.Context())
			require.NoError(t, err)
			require.Len(t, current.Fields, 2)
			document, err := client.Collection(name).Document("one").Retrieve(t.Context())
			require.NoError(t, err)
			require.Equal(t, "world", document["meta_title"])
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
	}})
}

type collectionPlanCheckFunc func()

func (check collectionPlanCheckFunc) CheckPlan(_ context.Context, _ plancheck.CheckPlanRequest, _ *plancheck.CheckPlanResponse) {
	check()
}
