package provider_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccKeyPermissionsAndRotation(t *testing.T) {
	t.Parallel()

	prefix := "keytest_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	firstSecret := "first_" + acctest.RandStringFromCharSet(32, acctest.CharSetAlphaNum)
	secondSecret := "second_" + acctest.RandStringFromCharSet(32, acctest.CharSetAlphaNum)
	config := func(secret string) string {
		return fmt.Sprintf(`resource "typesense_collection" "allowed" {
 name = "%[1]s_posts"
 fields = [{name = "title", type = "string"}]
}
resource "typesense_collection" "denied" {
 name = "other_%[1]s"
 fields = [{name = "title", type = "string"}]
}
resource "typesense_key" "test" {
 description = "search permission and rotation test"
 actions = ["documents:search"]
 collections = ["%[1]s_.*"]
 value = %[2]q
 lifecycle { create_before_destroy = true }
}
resource "typesense_key" "expired" {
 description = "expired key metadata remains available"
 actions = ["documents:search"]
 collections = ["*"]
 expires_at = 0
 autodelete = false
}
resource "typesense_key" "metadata" {
 description = "global key metadata permissions"
 actions = ["keys:list", "keys:get"]
 collections = ["no_matching_collection"]
}
`, prefix, secret)
	}
	check := func(secret string) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			assert.Equal(t, secret, state.RootModule().Resources["typesense_key.test"].Primary.Attributes["value"])
			keyHTTPStatus(t, secret, http.MethodGet, "/collections/"+prefix+"_posts/documents/search?q=*&query_by=title", "", http.StatusOK)
			keyHTTPStatus(t, secret, http.MethodGet, "/collections/other_"+prefix+"/documents/search?q=*&query_by=title", "", http.StatusUnauthorized)
			keyHTTPStatus(t, secret, http.MethodPost, "/collections/"+prefix+"_posts/documents", `{"title":"forbidden"}`, http.StatusUnauthorized)
			keyHTTPStatus(t, secret, http.MethodGet, "/keys", "", http.StatusUnauthorized)

			expired := state.RootModule().Resources["typesense_key.expired"].Primary.Attributes
			keyHTTPStatus(t, expired["value"], http.MethodGet, "/collections/"+prefix+"_posts/documents/search?q=*&query_by=title", "", http.StatusUnauthorized)

			metadata := state.RootModule().Resources["typesense_key.metadata"].Primary.Attributes
			keyHTTPStatus(t, metadata["value"], http.MethodGet, "/keys", "", http.StatusOK)
			keyHTTPStatus(t, metadata["value"], http.MethodGet, "/keys/"+expired["id"], "", http.StatusOK)
			keyHTTPStatus(t, metadata["value"], http.MethodDelete, "/keys/"+expired["id"], "", http.StatusUnauthorized)

			return nil
		}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(firstSecret), Check: check(firstSecret)},
			{Config: config(secondSecret), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace), Check: resource.ComposeTestCheckFunc(check(secondSecret), func(_ *terraform.State) error {
				keyHTTPStatus(t, firstSecret, http.MethodGet, "/collections/"+prefix+"_posts/documents/search?q=*&query_by=title", "", http.StatusUnauthorized)

				return nil
			})},
		},
		CheckDestroy: func(_ *terraform.State) error {
			keyHTTPStatus(t, secondSecret, http.MethodGet, "/collections/"+prefix+"_posts/documents/search?q=*&query_by=title", "", http.StatusUnauthorized)

			return nil
		},
	})
}

//nolint:gosec // Acceptance requests intentionally target the explicitly configured test server.
func keyHTTPStatus(t *testing.T, secret, method, path, body string, status int) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, os.Getenv("TYPESENSE_URL")+path, strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("X-Typesense-Api-Key", secret)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	require.NoError(t, err)

	defer response.Body.Close()

	_, err = io.Copy(io.Discard, response.Body)
	require.NoError(t, err)
	assert.Equal(t, status, response.StatusCode, "%s %s", method, path)
}
