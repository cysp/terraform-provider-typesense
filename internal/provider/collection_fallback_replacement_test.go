package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionFallbackWholeCollectionReplacement(t *testing.T) {
	for _, mode := range []string{"replace flag", "replace_triggered_by"} {
		t.Run(mode, func(t *testing.T) {
			var creates, deletes, patches atomic.Int32

			collections := map[string]api.CollectionResponse{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				switch req.Method {
				case http.MethodPost:
					creates.Add(1)
				case http.MethodDelete:
					deletes.Add(1)
				case http.MethodPatch:
					patches.Add(1)
				}

				handleLocalCollectionRequest(t, collections, w, req)
			}))
			t.Cleanup(server.Close)

			config := func(fieldType string) string {
				lifecycle := ""
				if mode == "replace_triggered_by" {
					lifecycle = `lifecycle { replace_triggered_by = [terraform_data.trigger] }`
				}

				return providerConfig(server.URL) + fmt.Sprintf(`
resource "terraform_data" "trigger" { input = %q }
resource "typesense_collection" "test" {
 name = "fallback_replacement"
 fields = [{name=".*",type=%q}]
 %s
}`, fieldType, fieldType, lifecycle)
			}
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: config("auto")},
					{
						Config: config("string"),
						PreConfig: func() {
							if mode == "replace flag" {
								t.Setenv("TF_CLI_ARGS_plan", "-replace=typesense_collection.test")
							}
						},
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionReplace)}},
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.type", "string"),
							func(_ *terraform.State) error {
								if mode == "replace flag" {
									t.Setenv("TF_CLI_ARGS_plan", "")
								}

								require.Equal(t, int32(2), creates.Load())
								require.Equal(t, int32(1), deletes.Load())
								require.Zero(t, patches.Load())

								return nil
							},
						),
					},
				},
			})
		})
	}
}

func TestCollectionFallbackInPlaceChangeSendsNoMutation(t *testing.T) {
	t.Parallel()

	var patches atomic.Int32

	collections := map[string]api.CollectionResponse{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPatch {
			patches.Add(1)
		}

		handleLocalCollectionRequest(t, collections, w, req)
	}))
	t.Cleanup(server.Close)

	config := func(fieldType string) string {
		return providerConfig(server.URL) + fmt.Sprintf(`
resource "typesense_collection" "test" {
 name = "fallback_in_place"
 fields = [{name=".*",type=%q}]
}`, fieldType)
	}
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("auto")},
			{Config: config("string"), ExpectError: regexp.MustCompile(`cannot replace an existing \.\* fallback in one schema alteration`)},
			{Config: config("auto"), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
		},
	})
	require.Zero(t, patches.Load())
}
