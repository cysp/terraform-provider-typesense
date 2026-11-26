package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionResourceDefaultSortingFieldAlteration(t *testing.T) {
	requireLiveAcceptance(t)
	t.Parallel()

	name := "testacc_" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	client, err := api.NewClientWithResponses(os.Getenv("TYPESENSE_URL"), api.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	require.NoError(t, err)

	config := func(attribute, rank string) string {
		return fmt.Sprintf(`resource "typesense_collection" "test" {
 name=%q
 %s
 fields=[{name="title",type="string"}%s]
}`, name, attribute, rank)
	}

	const (
		configured = `default_sorting_field="rank"`
		declared   = `,{name="rank",type="int64",sort=true}`
		reindexed  = `,{name="rank",type="int64",sort=true,facet=true}`
	)

	checkSort := func(want string) {
		current, err := client.GetCollectionWithResponse(t.Context(), name)
		require.NoError(t, err)
		require.NotNil(t, current.JSON200)
		require.Equal(t, new(want), current.JSON200.DefaultSortingField)
	}
	removed := config("", "")
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config(configured, declared)},
		{Config: config(configured+"\nlifecycle {ignore_changes=[fields]}", ""), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_collection.test", plancheck.ResourceActionNoop)}}},
		{Config: config("lifecycle {ignore_changes=[default_sorting_field]}", ""), PlanOnly: true, ExpectError: regexp.MustCompile("Missing default sorting field")},
		{Config: config(configured, reindexed), ExpectError: regexp.MustCompile("cannot reindex the default sorting field")},
		{Config: config("", reindexed), PreConfig: func() { checkSort("rank") }, ExpectError: regexp.MustCompile("cannot reindex the default sorting field")},
		{Config: config(configured, ""), PlanOnly: true, ExpectError: regexp.MustCompile("Missing default sorting field")},
		{Config: removed, PreConfig: func() { checkSort("rank") }, ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownValue("typesense_collection.test", tfjsonpath.New("default_sorting_field"), knownvalue.StringExact(""))}},
		{Config: removed, PlanOnly: true, PreConfig: func() { checkSort("") }},
	}})
}

func TestAccCollectionResourceDefaultSortingFieldSendsNoPatch(t *testing.T) {
	t.Parallel()

	collections := map[string]api.CollectionResponse{"posts": {Name: "posts", DefaultSortingField: new("rank"), Fields: []api.Field{{Name: "rank", Type: "int64", Sort: new(true)}, {Name: "title", Type: "string"}}}}

	var patches atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patches.Add(1)
			http.Error(w, "unexpected collection PATCH", http.StatusInternalServerError)

			return
		}

		handleLocalCollectionRequest(t, collections, w, r)
	}))
	t.Cleanup(server.Close)

	config := func(attribute, rank string) string {
		return providerConfig(server.URL) + fmt.Sprintf(`resource "typesense_collection" "test" {
 name="posts"
 %s
 fields=[{name="title",type="string"}%s]
}`, attribute, rank)
	}

	const (
		configured = `default_sorting_field="rank"`
		declared   = `,{name="rank",type="int64",sort=true}`
	)

	late := config(configured, `,{name=terraform_data.rename.output,type="int64",sort=true}`) + `
resource "terraform_data" "rename" {
 input = "other"
}`
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config(configured, declared), ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: "posts", ImportStatePersist: true},
		{Config: config("", `,{name="rank",type="int64",sort=true,facet=true}`), ExpectError: regexp.MustCompile("cannot reindex the default sorting field")},
		{Config: config("lifecycle {ignore_changes=[default_sorting_field]}", ""), ExpectError: regexp.MustCompile("Missing default sorting field")},
		{Config: late, ExpectError: regexp.MustCompile("Missing default sorting field")},
	}})
	require.Zero(t, patches.Load())
}
