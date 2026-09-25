package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccCollectionFieldOptions(t *testing.T) {
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := func(titleOptions, scoreOptions, vectorOptions string) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
  name = %q
  fields = [
    { name = "title", type = "string"%s },
    { name = "score", type = "int64"%s },
    { name = "embedding", type = "float[]", num_dim = 3, optional = true%s },
  ]
}`, name, titleOptions, scoreOptions, vectorOptions)
	}

	explicit := config(`, stem = true, token_separators = ["-"], symbols_to_index = ["+"]`,
		`, sort = false, range_index = true, store = false`, `, vec_dist = "ip"`)
	defaults := config("", "", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: explicit, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.stem", "true"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.token_separators.0", "-"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.symbols_to_index.0", "+"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.sort", "false"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.range_index", "true"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.store", "false"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.2.vec_dist", "ip"),
			)},
			{Config: defaults, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionUpdate),
			}}, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.stem", "false"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.token_separators.#", "0"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.0.symbols_to_index.#", "0"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.sort", "true"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.range_index", "false"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.1.store", "true"),
				resource.TestCheckResourceAttr("typesense_collection.test", "fields.2.vec_dist", "cosine"),
			)},
			{Config: defaults, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop),
			}}},
		},
	})
}
