package provider_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestProviderMutationNotReplayed(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			var requests, redirected atomic.Int32

			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				redirected.Add(1)
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			t.Cleanup(target.Close)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "/keys", req.URL.Path)
				requests.Add(1)
				w.Header().Set("Location", target.URL+"/keys")
				w.WriteHeader(status)
			}))
			t.Cleanup(server.Close)
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: providerConfig(server.URL) + `resource "typesense_key" "test" {
					 description = "transport test"
					 actions = ["documents:search"]
					 collections = ["*"]
					}`,
					ExpectError: regexp.MustCompile("Error creating key"),
				}},
			})
			assert.EqualValues(t, 1, requests.Load())
			assert.Zero(t, redirected.Load())
		})
	}
}
