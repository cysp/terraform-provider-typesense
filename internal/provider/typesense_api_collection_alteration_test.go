package provider_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAccCollectionResourceReindexFieldProperties(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		unknown bool
	}{
		{name: "preserve"},
		{name: "unrecognized", unknown: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			collections := map[string]api.CollectionResponse{}

			var patches [][]byte

			server := httptest.NewServer(fieldPropertiesHandler(t, collections, test.unknown, &patches))
			t.Cleanup(server.Close)
			config := providerConfig(server.URL) + `resource "typesense_collection" "test" {
 name="posts"
 fields=[{name="author",type="string",reference="authors.id",facet=false}]
}`

			change := resource.TestStep{Config: strings.Replace(config, "facet=false", "facet=true", 1)}
			if test.unknown {
				change.ExpectError = regexp.MustCompile("future_property")
			}

			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config}, change}})

			if test.unknown {
				require.Empty(t, patches)

				return
			}

			require.Len(t, patches, 1)

			var payload struct {
				Fields []map[string]json.RawMessage `json:"fields"`
			}
			require.NoError(t, json.Unmarshal(patches[0], &payload))
			require.Len(t, payload.Fields, 2)

			for property, expected := range map[string]string{"async_reference": "true", "cascade_delete": "false", "truncate_len": "7", "facet": "true"} {
				require.Equal(t, expected, string(payload.Fields[1][property]), property)
			}
		})
	}
}

func fieldPropertiesHandler(t *testing.T, collections map[string]api.CollectionResponse, unknown bool, patches *[][]byte) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPatch {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}

			*patches = append(*patches, body)
			req.Body = io.NopCloser(bytes.NewReader(body))
		}

		if req.Method != http.MethodGet {
			handleLocalCollectionRequest(t, collections, w, req)

			return
		}

		rec := httptest.NewRecorder()
		handleLocalCollectionRequest(t, collections, rec, req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.Code)

		if rec.Code != http.StatusOK {
			_, _ = w.Write(rec.Body.Bytes())

			return
		}

		var (
			result map[string]json.RawMessage
			fields []map[string]json.RawMessage
		)

		err := json.Unmarshal(rec.Body.Bytes(), &result)
		if err != nil {
			t.Error(err)

			return
		}

		err = json.Unmarshal(result["fields"], &fields)
		if err != nil {
			t.Error(err)

			return
		}

		fields[0]["async_reference"] = json.RawMessage("true")
		fields[0]["cascade_delete"] = json.RawMessage("false")

		fields[0]["truncate_len"] = json.RawMessage("7")
		if unknown {
			fields[0]["future_property"] = json.RawMessage("true")
		}

		result["fields"], _ = json.Marshal(fields)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func TestAccCollectionResourceAmbiguousFieldSource(t *testing.T) {
	t.Parallel()

	for _, firstType := range []string{"auto", "string"} {
		t.Run(firstType, func(t *testing.T) {
			t.Parallel()

			fields := []api.Field{
				{Name: "phrase", Type: "auto", Optional: new(true), Store: new(true)},
				{Name: "phrase", Type: "string", Optional: new(true), Store: new(false)},
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
