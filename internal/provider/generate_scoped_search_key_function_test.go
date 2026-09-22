package provider_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

// The decoded scoped-key payload starts after the 44-byte base64 digest and four-byte parent prefix.
func TestGenerateScopedSearchKeyFunction(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, params, json string }{
		{"empty", `{}`, `{}`},
		{"collections", `{list=tolist([1,2]),empty_list=tolist([]),empty_set=toset([]),empty_map=tomap({}),null_string=tostring(null),null_list=tolist(null),null_map=tomap(null)}`, `{"empty_list":[],"empty_map":{},"empty_set":[],"list":[1,2],"null_list":null,"null_map":null,"null_string":null}`},
		{"numbers", `{negative=-9007199254740993,fraction=-0.125,small=0.00000000000000000001}`, `{"fraction":-0.125,"negative":-9007199254740993,"small":0.00000000000000000001}`},
		{"map", `tomap({filter_by="tenant_id:=one"})`, `{"filter_by":"tenant_id:=one"}`},
		{"nested", `{filter_by="tenant_id:=one",limit_hits=3,enabled=true,missing=null,nested={empty=[],nothing={},mixed=["a",2,false,null]},set=toset(["b","a"]),large=9007199254740993,decimal=0.1}`, `{"decimal":0.1,"enabled":true,"filter_by":"tenant_id:=one","large":9007199254740993,"limit_hits":3,"missing":null,"nested":{"empty":[],"mixed":["a",2,false,null],"nothing":{}},"set":["a","b"]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
				Config:            fmt.Sprintf(`output "params" { value=substr(base64decode(provider::typesense::generate_scoped_search_key("abcdef",%s)),48,-1) }`, test.params),
				ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownOutputValue("params", knownvalue.StringExact(test.json))},
			}}})
		})
	}
}

func TestGenerateScopedSearchKeyFunctionVector(t *testing.T) {
	t.Parallel()

	expected := base64.StdEncoding.EncodeToString([]byte(`iy27OvahxUoH+TD/dNy4fNYc6S7Zjt+W/v0bKs6ugcE=abcd{"another":["1",2],"scope":"foo","something":"123","yetanother":{"a":1,"b":[2,"3"]}}`))
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config:            `output "key" { value=provider::typesense::generate_scoped_search_key("abcdef",{scope="foo",something="123",another=["1",2],yetanother={a=1,b=[2,"3"]}}) }`,
		ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownOutputValue("key", knownvalue.StringExact(expected))},
	}}})
}

func TestGenerateScopedSearchKeyFunctionInvalid(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, key, params, pattern string }{
		{"short", `"abc"`, `{}`, "at least four bytes"},
		{"empty_key", `""`, `{}`, "at least four bytes"},
		{"null_key", `null`, `{}`, "must not be null|Invalid function argument"},
		{"null_params", `"abcdef"`, `null`, "must not be null|Invalid function argument"},
		{"list", `"abcdef"`, `[]`, `must be an object or\s+map`},
		{"string", `"abcdef"`, `"wrong"`, `must be an object or\s+map`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
				Config: fmt.Sprintf(`output "key" { value=provider::typesense::generate_scoped_search_key(%s,%s) }`, test.key, test.params), ExpectError: regexp.MustCompile(test.pattern),
			}}})
		})
	}
}

func TestGenerateScopedSearchKeyFunctionUnknown(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config: `resource "terraform_data" "input" { input={key="abcdef",tenant="one"} }
output "params" { value=substr(base64decode(provider::typesense::generate_scoped_search_key(terraform_data.input.output.key,{filter_by="tenant_id:=${terraform_data.input.output.tenant}"})),48,-1) }
output "nested_unknown" { value=substr(base64decode(provider::typesense::generate_scoped_search_key("abcdef",{filter_by="tenant_id:=${terraform_data.input.output.tenant}"})),48,-1) }`,
		ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownOutputValue("params", knownvalue.StringExact(`{"filter_by":"tenant_id:=one"}`)), statecheck.ExpectKnownOutputValue("nested_unknown", knownvalue.StringExact(`{"filter_by":"tenant_id:=one"}`))},
	}}})
}

func TestAccGenerateScopedSearchKeyFunction(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config: fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 fields=[{name="title",type="string"},{name="tenant_id",type="string"}]
}
resource "typesense_key" "test" {
 description="scoped search test"
 actions=["documents:search"]
 collections=[typesense_collection.test.name]
}
output "key" {
 value=provider::typesense::generate_scoped_search_key(typesense_key.test.value,{filter_by="tenant_id:=one"})
 sensitive=true
}`, name),
		Check: func(state *terraform.State) error {
			for _, tenant := range []string{"one", "two"} {
				_, err := client.Collection(name).Documents().Create(t.Context(), map[string]any{"id": tenant, "title": "hello", "tenant_id": tenant}, &api.DocumentIndexParameters{})
				require.NoError(t, err)
			}

			key, ok := state.RootModule().Outputs["key"].Value.(string)
			require.True(t, ok)

			unrestricted, err := client.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("hello"), QueryBy: new("title")})
			require.NoError(t, err)
			require.Equal(t, new(2), unrestricted.Found)
			require.Len(t, *unrestricted.Hits, 2)
			require.ElementsMatch(t, []any{"one", "two"}, []any{(*(*unrestricted.Hits)[0].Document)["tenant_id"], (*(*unrestricted.Hits)[1].Document)["tenant_id"]})

			scoped := typesense.NewClient(typesense.WithServer(os.Getenv("TYPESENSE_URL")), typesense.WithAPIKey(key))
			result, err := scoped.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("hello"), QueryBy: new("title")})
			require.NoError(t, err)
			require.Equal(t, new(1), result.Found)
			require.Len(t, *result.Hits, 1)
			require.Equal(t, "one", (*(*result.Hits)[0].Document)["tenant_id"])

			for _, test := range []struct {
				filter string
				count  int
			}{{"tenant_id:=two", 0}, {"tenant_id:=[one,two]", 1}} {
				result, err = scoped.Collection(name).Documents().Search(t.Context(), &api.SearchCollectionParams{Q: new("hello"), QueryBy: new("title"), FilterBy: new(test.filter)})
				require.NoError(t, err)
				require.Equal(t, new(test.count), result.Found)
				require.Len(t, *result.Hits, test.count)

				for _, hit := range *result.Hits {
					require.Equal(t, "one", (*hit.Document)["tenant_id"])
				}
			}

			_, err = scoped.Collection(name).Documents().Create(t.Context(), map[string]any{"id": "forbidden", "title": "hello", "tenant_id": "one"}, &api.DocumentIndexParameters{})

			var apiError *typesense.HTTPError
			require.ErrorAs(t, err, &apiError)
			require.Equal(t, http.StatusUnauthorized, apiError.Status)

			return nil
		},
	}}})
}

func TestGenerateScopedSearchKeyFunctionMinimumParent(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config:            `output "key" { value=provider::typesense::generate_scoped_search_key("abcd",{}) }`,
		ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownOutputValue("key", knownvalue.StringExact("OFc2NHh4S0YwN0s0YWwvQ0ljZThyblFrTGQ4SXVKOG9UOG81UCtoa2JiOD1hYmNke30="))},
	}}})
}
