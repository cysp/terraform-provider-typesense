package provider_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionSchemaLifecycle(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name, declarations string
		document           map[string]any
	}{
		{"static", `{name="title",type="string"}, {name="obsolete",type="string",optional=true}`, map[string]any{"title": "Original", "obsolete": "retained"}},
		{"catchall", `{name="title",type="string"}, {name=".*",type="auto",optional=true}`, map[string]any{"title": "Original", "count": 42}},
		{"literal_auto", `{name="title",type="string"}, {name="count",type="auto",optional=true}`, map[string]any{"title": "Original", "count": 42}},
		{"literal_string", `{name="title",type="string"}, {name="tags",type="string*",optional=true}`, map[string]any{"title": "Original", "tags": []string{"one"}}},
		{"nested", `{name="title",type="string"}, {name="person",type="object",optional=true}`, map[string]any{"title": "Original", "person": map[string]any{"age": 42, "name": "Ada"}}},
		{"typed_numeric_pattern", `{name="title",type="string"}, {name="score_.*",type="int64",optional=true}`, map[string]any{"title": "Original", "score_a": 42}},
		{"object_pattern", `{name="title",type="string"}, {name="person_.*",type="object",optional=true}`, map[string]any{"title": "Original", "person_a": map[string]any{"age": 42}}},
		{"pattern", `{name="title",type="string"}, {name="meta_.*",type="auto",optional=true}`, map[string]any{"title": "Original", "meta_count": 42}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
			config := func(fields string) string {
				return fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
    enable_nested_fields = true
    fields=[%s]
   }`, name, fields)
			}
			original := config(scenario.declarations)
			final := config(`{name="title",type="string",facet=true}, {name="added",type="string",optional=true}`)
			scenario.document["id"] = "one"

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: original},
					{
						PreConfig: func() {
							_, err := client.Collection(name).Documents().Create(context.Background(), scenario.document, &api.DocumentIndexParameters{})
							require.NoError(t, err)
						}, Config: final, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}},
						ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownValue("typesense_collection.test", tfjsonpath.New("num_documents"), knownvalue.Int64Exact(1))},
					},
					{Config: final, PreConfig: func() {
						document, err := client.Collection(name).Document("one").Retrieve(context.Background())
						require.NoError(t, err)
						require.Equal(t, "Original", document["title"])

						current, err := client.Collection(name).Retrieve(context.Background())
						require.NoError(t, err)

						names := make([]string, 0, len(current.Fields))
						for _, field := range current.Fields {
							names = append(names, field.Name)
						}

						slices.Sort(names)
						require.Equal(t, []string{"added", "title"}, names)
					}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
				},
			})
		})
	}
}

func TestAccCollectionExternalDrift(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	config := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 fields=[{name="title",type="string",optional=true}, {name=".*",type="auto",optional=true}]
 }`, name)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config},
		{Config: config, PreConfig: func() {
			_, err := client.Collection(name).Update(context.Background(), &api.CollectionUpdateSchema{Fields: []api.Field{
				{Name: "title", Drop: new(true)},
				{Name: "title", Type: "string", Optional: new(true), Facet: new(true), Store: new(false)},
				{Name: "rogue.*", Type: "string", Optional: new(true)},
				{Name: "extra", Type: "string", Optional: new(true), Facet: new(true)},
			}})
			require.NoError(t, err)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
		{Config: config, PreConfig: func() {
			current, err := client.Collection(name).Retrieve(context.Background())
			require.NoError(t, err)
			require.Len(t, current.Fields, 2)
			index := slices.IndexFunc(current.Fields, func(field api.Field) bool { return field.Name == "title" })
			require.NotEqual(t, -1, index)
			require.NotNil(t, current.Fields[index].Store)
			require.False(t, *current.Fields[index].Store)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
		{Config: strings.ReplaceAll(config, `type="string"`, `type="string[]"`), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
		{Config: strings.ReplaceAll(config, `type="string"`, `type="string[]"`), PreConfig: func() {
			current, err := client.Collection(name).Retrieve(context.Background())
			require.NoError(t, err)

			index := slices.IndexFunc(current.Fields, func(field api.Field) bool { return field.Name == "title" })
			require.NotEqual(t, -1, index)
			require.Equal(t, "string[]", current.Fields[index].Type)
			require.NotNil(t, current.Fields[index].Store)
			require.False(t, *current.Fields[index].Store)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
	}})
}

func TestAccCollectionExplicitNestedFields(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	config := func(parentFacet bool) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 enable_nested_fields=true
 fields=[
  {name="person",type="object",optional=true,facet=%t},
  {name="person.title",type="string",optional=true,facet=true},
  {name="person.age",type="int64",optional=true,sort=true}
 ]
}`, name, parentFacet)
	}
	original, changed := config(false), config(true)
	empty := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 enable_nested_fields=true
 fields=[]
}`, name)
	document := map[string]any{"id": "one", "person": map[string]any{"title": "Editor", "age": float64(42)}}

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: original},
		{Config: original, PreConfig: func() {
			_, err := client.Collection(name).Documents().Create(t.Context(), document, &api.DocumentIndexParameters{})
			require.NoError(t, err)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
		{Config: changed, ExpectError: regexp.MustCompile("No schema alteration was sent"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
		{Config: original, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
		{Config: empty},
		{Config: changed},
		{Config: changed, PreConfig: func() {
			stored, err := client.Collection(name).Document("one").Retrieve(t.Context())
			require.NoError(t, err)
			require.Equal(t, document, stored)
			result, err := client.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("Editor"), QueryBy: new("person.title")})
			require.NoError(t, err)
			require.Equal(t, 1, *result.Found)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
	}})
}

func TestAccCollectionExplicitFieldRediscovery(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	config := func(explicit string) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 fields=[{name=".*",type="auto",optional=true}, %s]
 }`, name, explicit)
	}
	remaining := config("")
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config(`{name="title",type="string",facet=true}`)},
		{Config: remaining, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
		{Config: remaining, PreConfig: func() {
			_, err := client.Collection(name).Documents().Create(context.Background(), map[string]any{"id": "one", "title": "Rediscovered"}, &api.DocumentIndexParameters{})
			require.NoError(t, err)
			collection, err := client.Collection(name).Retrieve(context.Background())
			require.NoError(t, err)

			index := slices.IndexFunc(collection.Fields, func(field api.Field) bool { return field.Name == "title" })
			require.NotEqual(t, -1, index)
			require.NotNil(t, collection.Fields[index].Facet)
			require.False(t, *collection.Fields[index].Facet)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
		{Config: remaining, PreConfig: func() {
			current, err := client.Collection(name).Retrieve(context.Background())
			require.NoError(t, err)
			require.Len(t, current.Fields, 1)
			require.Equal(t, ".*", current.Fields[0].Name)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
	}})
}

func TestAccCollectionRemovalRetainsExplicitField(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name, parent, kind, child string
		document                  map[string]any
	}{
		{name: "object_regex", parent: "person_.*", kind: "object", child: "person_a.title"},
		{name: "object_array_regex", parent: "person_.*", kind: "object[]", child: "person_a.title"},
		{name: "dotted_scalar", parent: "a.b", kind: "string", child: "a.b.c"},
		{name: "auto_regex", parent: "a.b", kind: "auto", child: "aXb"},
		{name: "string_regex", parent: "a.b", kind: "string*", child: "aXb"},
		{name: "bracket_regex", parent: "tags[0]", kind: "auto", child: "tags0"},
		{name: "developed_auto_object", parent: "person", kind: "auto", child: "person.title", document: map[string]any{"id": "one", "person": map[string]any{"title": "Editor"}}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
			config := func(rule string) string {
				return fmt.Sprintf(`resource "typesense_collection" "test" {
    name=%q
    enable_nested_fields=true
    fields=[%s {name=%q,type="string",optional=true,facet=true}]
   }`, name, rule, scenario.child)
			}
			original := config(fmt.Sprintf(`{name=%q,type=%q,optional=true},`, scenario.parent, scenario.kind))
			desired := config("")

			var alterationError *regexp.Regexp
			if scenario.kind == "auto" || scenario.kind == "string*" || strings.Contains(scenario.parent, ".*") {
				alterationError = regexp.MustCompile("No schema alteration was sent")
			}

			steps := []resource.TestStep{
				{Config: original},
				{Config: desired, ExpectError: alterationError, PreConfig: func() {
					if scenario.document != nil {
						_, err := client.Collection(name).Documents().Create(context.Background(), scenario.document, &api.DocumentIndexParameters{})
						require.NoError(t, err)
					}
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
			}
			if alterationError != nil {
				empty := fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 enable_nested_fields=true
 fields=[]
}`, name)
				steps = append(steps, resource.TestStep{Config: empty}, resource.TestStep{Config: desired})
			}

			steps = append(steps, resource.TestStep{Config: desired, PreConfig: func() {
				if scenario.document != nil {
					document, err := client.Collection(name).Document("one").Retrieve(context.Background())
					require.NoError(t, err)
					require.Equal(t, scenario.document, document)
				}
			}, ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownValue("typesense_collection.test", tfjsonpath.New("fields"), knownvalue.ListSizeExact(1))}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}})
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: steps})
		})
	}
}

func TestAccCollectionExpansionAdoption(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name, firstType string
		imported        bool
	}{
		{"created_concrete", "int64", false},
		{"imported_concrete", "int64", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
			config := func(explicit string) string {
				return fmt.Sprintf(`resource "typesense_collection" "test" {
     name=%q
     fields=[{name=".*",type="auto",optional=true}, %s]
    }`, name, explicit)
			}
			original := config("")
			adopted := config(fmt.Sprintf(`{name="score",type=%q,optional=true,sort=%t}`, scenario.firstType, scenario.firstType == "int64"))
			faceted := config(`{name="score",type="int64",optional=true,sort=true,facet=true}`)
			converted := config(`{name="score",type="float",optional=true,sort=true,facet=true}`)
			steps := make([]resource.TestStep, 0, 12)

			steps = append(steps,
				resource.TestStep{Config: original},
				resource.TestStep{Config: adopted, PreConfig: func() {
					_, err := client.Collection(name).Documents().Create(context.Background(), map[string]any{"id": "one", "score": 42}, &api.DocumentIndexParameters{})
					require.NoError(t, err)
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			)
			if scenario.imported {
				steps = []resource.TestStep{{
					Config: original, ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: name, ImportStatePersist: true,
					PreConfig: func() {
						_, err := client.Collections().Create(context.Background(), &api.CollectionSchema{Name: name, Fields: []api.Field{{Name: ".*", Type: "auto", Optional: new(true)}}})
						require.NoError(t, err)
						_, err = client.Collection(name).Documents().Create(context.Background(), map[string]any{"id": "one", "score": 42}, &api.DocumentIndexParameters{})
						require.NoError(t, err)
					},
				}}
			}

			steps = append(steps, resource.TestStep{Config: adopted, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}})
			for _, desired := range []string{faceted, converted} {
				steps = append(steps,
					resource.TestStep{Config: desired, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate)}}},
					resource.TestStep{Config: desired, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
				)
			}

			steps = append(steps,
				resource.TestStep{Config: original},
				resource.TestStep{Config: original, PreConfig: func() {
					current, err := client.Collection(name).Retrieve(context.Background())
					require.NoError(t, err)
					require.Len(t, current.Fields, 1)
					require.Equal(t, ".*", current.Fields[0].Name)

					document, err := client.Collection(name).Document("one").Retrieve(context.Background())
					require.NoError(t, err)
					require.InDelta(t, 42, document["score"], 0)
				}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: steps})
		})
	}
}
