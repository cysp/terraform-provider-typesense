package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccKeyAttributeReplacement(t *testing.T) {
	t.Parallel()

	description := "key attribute replacement"
	actions := `["documents:search"]`
	collections := `["posts"]`
	expiration := ""
	value := ""
	config := func() string {
		return fmt.Sprintf(`resource "typesense_key" "test" {
 description = %q
 actions = %s
 collections = %s
 %s
 %s
}`, description, actions, collections, expiration, value)
	}
	steps := make([]resource.TestStep, 0, 7)
	steps = append(steps, resource.TestStep{Config: config()})

	description = "updated description"

	steps = append(steps, resource.TestStep{Config: config(), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace)})

	// Preserve duplicate actions: deduplication can change scoped-parent eligibility.
	actions = `["documents:search", "documents:search"]`

	steps = append(steps, resource.TestStep{Config: config(), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace)})

	collections = `["posts", "posts_.*", "posts"]`

	steps = append(steps, resource.TestStep{Config: config(), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace)})

	expiration = "expires_at = 4102444800"

	steps = append(steps, resource.TestStep{Config: config(), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace)})

	value = fmt.Sprintf("value = %q", acctest.RandStringFromCharSet(32, acctest.CharSetAlphaNum))

	steps = append(steps, resource.TestStep{Config: config(), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace)})
	steps = append(steps, resource.TestStep{Config: config(), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionNoop)})

	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: steps})
}
