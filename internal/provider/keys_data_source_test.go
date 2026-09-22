package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeysDataSourceSchema(t *testing.T) {
	t.Parallel()

	var response datasource.SchemaResponse
	provider.NewKeysDataSource().Schema(t.Context(), datasource.SchemaRequest{}, &response)
	require.Empty(t, response.Diagnostics)
	keys, ok := response.Schema.Attributes["keys"].(schema.ListNestedAttribute)
	require.True(t, ok)

	attributes := keys.NestedObject.Attributes
	require.NotContains(t, attributes, "value")

	for name, attribute := range attributes {
		require.True(t, attribute.IsComputed(), name)
		require.False(t, attribute.IsOptional(), name)
		require.False(t, attribute.IsRequired(), name)
	}
}

func TestKeysDataSource(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name         string
		status       int
		body         string
		expected     knownvalue.Check
		errorPattern string
	}{
		{name: "empty", status: http.StatusOK, body: `{"keys":[]}`, expected: knownvalue.ListExact([]knownvalue.Check{})},
		{name: "metadata", status: http.StatusOK, body: `{"keys":[{"id":9,"description":"second","actions":["documents:search"],"collections":["posts"],"expires_at":123,"value_prefix":"seco"},{"id":3,"description":"first","actions":["keys:list"],"collections":["*"],"expires_at":456,"value_prefix":"firs","value":"must-not-be-exposed"}]}`, expected: knownvalue.ListExact([]knownvalue.Check{
			knownvalue.ObjectExact(map[string]knownvalue.Check{"id": knownvalue.Int64Exact(3), "description": knownvalue.StringExact("first"), "actions": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("keys:list")}), "collections": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("*")}), "expires_at": knownvalue.Int64Exact(456), "value_prefix": knownvalue.StringExact("firs")}),
			knownvalue.ObjectExact(map[string]knownvalue.Check{"id": knownvalue.Int64Exact(9), "description": knownvalue.StringExact("second"), "actions": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("documents:search")}), "collections": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("posts")}), "expires_at": knownvalue.Int64Exact(123), "value_prefix": knownvalue.StringExact("seco")}),
		})},
		{name: "forbidden", status: http.StatusForbidden, body: `{"message":"Forbidden"}`, errorPattern: "Error retrieving keys"},
		{name: "server_error", status: http.StatusServiceUnavailable, body: `{"message":"Unavailable"}`, errorPattern: "Error retrieving keys"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodGet, req.Method)
				assert.Equal(t, "/keys", req.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, err := fmt.Fprint(w, test.body)
				assert.NoError(t, err)
			}))
			t.Cleanup(server.Close)

			step := resource.TestStep{Config: providerConfig(server.URL) + `data "typesense_keys" "test" {}`}
			if test.errorPattern != "" {
				step.ExpectError = regexp.MustCompile(test.errorPattern)
			} else {
				step.ConfigStateChecks = []statecheck.StateCheck{statecheck.ExpectKnownValue("data.typesense_keys.test", tfjsonpath.New("keys"), test.expected)}
			}

			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{step}})
		})
	}
}

func TestAccKeysDataSource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config: `resource "typesense_key" "test" {
 description="key listing test"
 actions=["documents:search"]
 collections=["posts"]
}
data "typesense_keys" "test" { depends_on=[typesense_key.test] }
output "listed" { value=one([for key in data.typesense_keys.test.keys:key if key.id==typesense_key.test.id]) }`,
		ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownOutputValue("listed", knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"description": knownvalue.StringExact("key listing test"),
			"actions":     knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("documents:search")}),
			"collections": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("posts")}),
			"expires_at":  knownvalue.NotNull(), "value_prefix": knownvalue.NotNull(),
		}))},
	}}})
}
