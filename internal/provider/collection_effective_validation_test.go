package provider_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionResourceEffectiveValidation(t *testing.T) {
	t.Parallel()

	const (
		nested = `resource "typesense_collection" "test" {
 name="posts"
 enable_nested_fields=true
 fields=[{name="person.score",type="auto",optional=false}]
}`
		ignored = `resource "typesense_collection" "test" {
 name="posts"
 fields=[{name=".*",type="auto"}]
 timeouts={update="5m"}
 lifecycle { ignore_changes=[fields] }
}`
		computed = `resource "terraform_data" "flag" {input=false}
resource "typesense_collection" "test" {
 name="posts"
 fields=[{name=".*",type="auto",store=terraform_data.flag.output ? true : null}]
}`
	)
	for _, test := range []struct {
		name, initial, updated string
		patches                int
	}{
		{name: "observed nesting", initial: nested, updated: strings.Replace(strings.Replace(nested, "enable_nested_fields=true", "", 1), "optional=false}]", "optional=false},{name=\"title\",type=\"string\"}]", 1), patches: 1},
		{name: "ignored fields", initial: ignored, updated: strings.Replace(strings.Replace(ignored, `name=".*",type="auto"`, `name="title",type="string",store=false`, 1), "5m", "6m", 1)},
		{name: "unknown becomes null", initial: computed, updated: computed},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			collections := map[string]api.CollectionResponse{}
			patches := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPatch {
					patches++
				}

				handleLocalCollectionRequest(t, collections, w, req)
			}))
			t.Cleanup(server.Close)
			cfg := providerConfig(server.URL)
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: cfg + test.initial},
				{Config: cfg + test.updated},
				{Config: cfg + test.updated, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
			}})
			require.Equal(t, test.patches, patches)
		})
	}
}
