package provider_test

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionNativeDynamicExpansions(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name, fields string
		document     map[string]any
	}{
		{"typed_nested_string", `{name="price.*",type="string",optional=true}`, map[string]any{"price": map[string]any{"USD": "forty-two"}}},
		{"typed_nested_numeric", `{name="price.*",type="float",optional=true}`, map[string]any{"price": map[string]any{"USD": 42}}},
		{"literal_auto_regex", `{name="tags[0]",type="auto",optional=true}`, map[string]any{"tags0": "inferred"}},
		{"literal_string_regex", `{name="tags[0]",type="string*",optional=true}`, map[string]any{"tags0": "inferred"}},
		{"typed_fallback_under_object", `{name=".*",type="string",optional=true},{name="person",type="object",optional=true}`, map[string]any{"person": map[string]any{"age": 42}}},
		{"fallback_without_dynamic_match", `{name=".*",type="auto",optional=true},{name="other.*",type="string",optional=true,facet=true}`, map[string]any{"priceUSD": "forty-two"}},
		{"fallback_literal_dotted_key", `{name=".*",type="auto",optional=true},{name="person",type="object",optional=true,facet=true}`, map[string]any{"person.name": "Ada"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
			config := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 enable_nested_fields=true
 fields=[%s]
 }`, name, scenario.fields)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config},
				{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true, PreConfig: func() {
					scenario.document["id"] = "one"
					_, err := client.Collection(name).Documents().Create(context.Background(), scenario.document, &api.DocumentIndexParameters{})
					require.NoError(t, err)
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
			}})
		})
	}
}

func TestAccCollectionTypedNestedExpansionDrift(t *testing.T) {
	t.Parallel()

	for _, setting := range []string{"type", "facet"} {
		t.Run(setting, func(t *testing.T) {
			t.Parallel()

			name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
			config := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 enable_nested_fields=true
 fields=[{name="price.*",type="string",optional=true}]
 }`, name)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config},
				{Config: config, PreConfig: func() {
					_, err := client.Collection(name).Documents().Create(context.Background(), map[string]any{"id": "one", "price": map[string]any{"USD": "true"}}, &api.DocumentIndexParameters{})
					require.NoError(t, err)
					_, err = client.Collection(name).Document("one").Delete(context.Background())
					require.NoError(t, err)

					changed := api.Field{Name: "price.USD", Type: "string", Optional: new(true)}
					if setting == "type" {
						changed.Type = "bool"
					} else {
						changed.Facet = new(true)
					}

					_, err = client.Collection(name).Update(context.Background(), &api.CollectionUpdateSchema{Fields: []api.Field{{Name: "price.USD", Drop: new(true)}, changed}})
					require.NoError(t, err)
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
				{Config: config, PreConfig: func() {
					current, err := client.Collection(name).Retrieve(context.Background())
					require.NoError(t, err)
					require.False(t, slices.ContainsFunc(current.Fields, func(field api.Field) bool { return field.Name == "price.USD" }))
					require.False(t, slices.ContainsFunc(current.Fields, func(field api.Field) bool { return field.Name == "price" && field.Type == "object" }))
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			}})
		})
	}
}
