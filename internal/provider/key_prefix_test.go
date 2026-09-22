package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccKeyPrefixMetadata(t *testing.T) {
	t.Parallel()

	for _, length := range []int{3, 4, 5} {
		t.Run(fmt.Sprintf("%d_bytes", length), func(t *testing.T) {
			t.Parallel()

			secret := acctest.RandStringFromCharSet(length, acctest.CharSetAlphaNum)
			prefix := secret[:min(4, length)]

			// Typesense returns the first up to four bytes even when that is the
			// complete secret. Both metadata data sources must preserve that value.
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`resource "typesense_key" "test" {
 description = "server-provided prefix"
 actions = ["keys:list"]
 collections = ["*"]
 value = %q
}
data "typesense_key" "test" { id = typesense_key.test.id }
data "typesense_keys" "test" { depends_on = [typesense_key.test] }
output "listed_prefix" { value = one([for key in data.typesense_keys.test.keys : key.value_prefix if key.id == typesense_key.test.id]) }
`, secret),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("value"), knownvalue.StringExact(secret)),
						statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("value_prefix"), knownvalue.StringExact(prefix)),
						statecheck.ExpectKnownValue("data.typesense_key.test", tfjsonpath.New("value_prefix"), knownvalue.StringExact(prefix)),
						statecheck.ExpectKnownOutputValue("listed_prefix", knownvalue.StringExact(prefix)),
					},
				}},
			})
		})
	}
}
