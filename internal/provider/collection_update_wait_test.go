package provider //nolint:testpackage // Exercise Update state retention and private reconciliation.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionUpdateReadConvergence(t *testing.T) {
	t.Parallel()

	title := normalizeCollectionField(api.Field{Name: "title", Type: "string"})
	faceted := title
	faceted.Facet = new(true)
	wildcard := normalizeCollectionField(api.Field{Name: ".*", Type: "auto", Optional: new(true)})

	score := normalizeCollectionField(api.Field{Name: "score", Type: "int64", Optional: new(true), Sort: new(true)})
	for _, scenario := range []struct {
		name             string
		prior, desired   []api.Field
		timeout, failure string
		freshReads       []bool
	}{
		{"stale_then_converged", []api.Field{title}, []api.Field{faceted}, "5s", "", []bool{false, true, true}},
		{"intermittent_replica_lag", []api.Field{title}, []api.Field{faceted}, "5s", "", []bool{true, false, true, true}},
		{"covered_explicit_removal", []api.Field{wildcard, score}, []api.Field{wildcard}, "5s", "", []bool{false, true, true}},
		{"deadline", []api.Field{title}, []api.Field{faceted}, "250ms", "deadline", nil},
		{"unexpected_expansion", []api.Field{title}, []api.Field{faceted}, "250ms", "extra", []bool{true}},
		{"read_unauthorized", []api.Field{title}, []api.Field{faceted}, "5s", "unauthorized", nil},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			var mutations, verificationReads atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, "/collections/posts", req.URL.Path)
				w.Header().Set("Content-Type", "application/json")

				if req.Method == http.MethodPatch {
					mutations.Add(1)

					var patch api.CollectionUpdateSchema
					assert.NoError(t, json.NewDecoder(req.Body).Decode(&patch))
					assert.NotEmpty(t, patch.Fields)
					assert.NoError(t, json.NewEncoder(w).Encode(patch))

					return
				}

				assert.Equal(t, http.MethodGet, req.Method)

				fields := scenario.prior

				var count int32
				if mutations.Load() > 0 {
					count = verificationReads.Add(1)
				}

				if count > 0 && scenario.failure == "unauthorized" {
					http.Error(w, "verification authorization denied", http.StatusUnauthorized)

					return
				}

				if count > 0 && len(scenario.freshReads) > 0 && scenario.freshReads[min(int(count)-1, len(scenario.freshReads)-1)] {
					fields = scenario.desired
				}

				if count > 0 && scenario.failure == "extra" {
					fields = append(slices.Clone(fields), score)
				}

				assert.NoError(t, json.NewEncoder(w).Encode(api.CollectionResponse{Name: "posts", Fields: fields}))
			}))
			t.Cleanup(server.Close)
			client, err := api.NewClientWithResponses(server.URL, api.WithHTTPClient(server.Client()))
			require.NoError(t, err)

			collection := collectionResource{providerData: TypesenseProviderData{client: typesense.NewClient(typesense.WithAPIClient(client)), alter: make(chan struct{}, 1)}}

			prior := collectionUpdateModel(t, scenario.prior, scenario.timeout)
			desired := collectionUpdateModel(t, scenario.desired, scenario.timeout)
			schema := prior.ResourceSchema(t.Context())
			req := resource.UpdateRequest{State: tfsdk.State{Schema: schema}, Plan: tfsdk.Plan{Schema: schema}}
			require.Empty(t, req.State.Set(t.Context(), &prior))
			require.Empty(t, req.Plan.Set(t.Context(), &desired))
			// The Framework initializes Update's response from the prior state.
			resp := resource.UpdateResponse{State: req.State}
			collection.Update(t.Context(), req, &resp)
			require.EqualValues(t, 1, mutations.Load())

			if scenario.failure != "" {
				require.True(t, resp.Diagnostics.HasError())
				require.Equal(t, "Cannot verify collection update", resp.Diagnostics[0].Summary())
				require.Contains(t, resp.Diagnostics[0].Detail(), "No mutation was retried")
				require.True(t, resp.State.Raw.Equal(req.State.Raw), "failed verification must retain prior state")
				require.EqualValues(t, 1, verificationReads.Load(), "verification errors must not be retried")

				return
			}

			require.Empty(t, resp.Diagnostics)
			require.EqualValues(t, len(scenario.freshReads), verificationReads.Load(), "stale state must be rejected before consecutive matching observations")

			var final CollectionModel
			require.Empty(t, resp.State.Get(t.Context(), &final))
			require.Equal(t, desired.Fields, final.Fields)
			require.Equal(t, prior.Name, final.Name)
		})
	}
}

func collectionUpdateModel(t *testing.T, fields []api.Field, updateTimeout string) CollectionModel {
	t.Helper()

	model := CollectionModel{Timeouts: timeouts.Value{Object: types.ObjectValueMust(
		map[string]attr.Type{"create": types.StringType, "read": types.StringType, "update": types.StringType, "delete": types.StringType},
		map[string]attr.Value{"create": types.StringNull(), "read": types.StringNull(), "update": types.StringValue(updateTimeout), "delete": types.StringNull()},
	)}}
	require.Empty(t, model.ReadFromResponse(t.Context(), &api.CollectionResponse{Name: "posts", Fields: fields}))

	return model
}
