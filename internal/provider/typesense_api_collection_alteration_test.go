package provider_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionResourceReindexFieldProperties(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, unknown string
	}{
		{name: "preserve"},
		{name: "unknown member", unknown: "future_property"},
		{name: "case variant", unknown: "cascade_deletE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			field := map[string]any{
				"name": "author", "type": "string", "reference": "authors.id", "facet": false,
				"index": true, "infix": false, "locale": "", "optional": false, "sort": false,
				"async_reference": true, "cascade_delete": false, "truncate_len": 7,
			}
			if test.unknown != "" {
				field[test.unknown] = true
			}

			initial, err := json.Marshal(field)
			require.NoError(t, err)

			var (
				patches    [][]byte
				handlerErr error
			)

			server := httptest.NewServer(fieldPropertiesHandler(initial, &patches, &handlerErr))
			t.Cleanup(server.Close)
			config := providerConfig(server.URL) + `resource "typesense_collection" "test" {
 name="posts"
 fields=[{name="author",type="string",reference="authors.id",facet=true}]
}`

			change := resource.TestStep{Config: config}
			if test.unknown != "" {
				change.ExpectError = regexp.MustCompile(test.unknown)
			}

			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config, ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: "posts", ImportStatePersist: true},
				change,
			}})
			server.Close()
			require.NoError(t, handlerErr)

			if test.unknown != "" {
				require.Empty(t, patches)

				return
			}

			require.Len(t, patches, 1)

			expected, err := json.Marshal(map[string]any{"fields": []map[string]any{
				{"name": "author", "type": "", "drop": true},
				{
					"name": "author", "type": "string", "facet": true, "index": true, "infix": false,
					"locale": "", "optional": false, "sort": false, "reference": "authors.id",
					"async_reference": true, "cascade_delete": false, "truncate_len": 7,
				},
			}})
			require.NoError(t, err)
			require.JSONEq(t, string(expected), string(patches[0]))
		})
	}
}

var errExpectedFieldReindex = errors.New("expected a drop and re-add of the field")

func fieldPropertiesHandler(current json.RawMessage, patches *[][]byte, handlerErr *error) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if req.URL.Path != "/collections/posts" || current == nil {
			http.NotFound(w, req)

			return
		}

		switch req.Method {
		case http.MethodGet, http.MethodDelete:
			err := json.NewEncoder(w).Encode(map[string]any{"name": "posts", "fields": []json.RawMessage{current}})
			if err != nil {
				*handlerErr = err
			}

			if req.Method == http.MethodDelete {
				current = nil
			}
		case http.MethodPatch:
			body, err := io.ReadAll(req.Body)
			if err != nil {
				*handlerErr = err
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}

			*patches = append(*patches, body)

			var patch struct {
				Fields []json.RawMessage `json:"fields"`
			}

			err = json.Unmarshal(body, &patch)
			if err != nil || len(patch.Fields) != 2 {
				*handlerErr = errors.Join(errExpectedFieldReindex, err)

				http.Error(w, "invalid alteration", http.StatusBadRequest)

				return
			}

			current = patch.Fields[1]

			_, err = w.Write(body)
			if err != nil {
				*handlerErr = err
			}
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}
}

func TestAccCollectionResourceAmbiguousFieldSource(t *testing.T) {
	t.Parallel()

	for _, firstType := range []string{"auto", "string"} {
		t.Run(firstType, func(t *testing.T) {
			t.Parallel()

			fields := []api.Field{
				{Name: "phrase", Type: "auto", Optional: new(true), Store: new(true)},
				{Name: "phrase", Type: "string", Optional: new(true), Stem: new(true)},
			}
			if firstType == "string" {
				fields[0], fields[1] = fields[1], fields[0]
			}

			collections := map[string]api.CollectionResponse{"posts": {Name: "posts", Fields: fields}}

			var patches atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPatch {
					patches.Add(1)
				}

				handleLocalCollectionRequest(t, collections, w, req)
			}))
			t.Cleanup(server.Close)
			config := providerConfig(server.URL) + `resource "typesense_collection" "test" {
 name="posts"
 fields=[{name="phrase",type="string[]",optional=true}]
}`
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
				{Config: config, ResourceName: "typesense_collection.test", ImportState: true, ImportStateId: "posts", ImportStatePersist: true},
				{Config: config, ExpectError: regexp.MustCompile(`(?s)ambiguous.*"phrase".*auto.*string`)},
			}})
			require.Zero(t, patches.Load())
		})
	}
}
