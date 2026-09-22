package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"
)

func TestKeyAutodeleteLifecycle(t *testing.T) {
	t.Parallel()

	// Keep the HTTP oracle independent of provider and SDK key models.
	var mutex sync.Mutex

	keys := map[string]map[string]any{"99": {
		"id": 99, "description": "key autodelete lifecycle", "actions": []string{"documents:search"},
		"collections": []string{"posts"}, "expires_at": int64(64723363199), "autodelete": true, "value_prefix": "impo",
	}}
	nextID := 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		assert.Equal(t, "test", req.Header.Get("X-Typesense-Api-Key"))

		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/keys":
			var key map[string]any

			err := json.NewDecoder(req.Body).Decode(&key)
			if !assert.NoError(t, err) {
				w.WriteHeader(http.StatusBadRequest)

				return
			}

			if _, exists := key["autodelete"]; !exists {
				key["autodelete"] = false
			}

			if _, exists := key["expires_at"]; !exists {
				key["expires_at"] = int64(64723363199)
			}

			if _, exists := key["value"]; !exists {
				key["value"] = fmt.Sprintf("generated-key-%d", nextID)
			}

			key["id"] = nextID
			keys[strconv.Itoa(nextID)] = key
			nextID++

			writeJSON(t, w, http.StatusCreated, key)
			// Reads expose only metadata, while Terraform must retain the secret.
			secret, ok := key["value"].(string)
			assert.True(t, ok)

			key["value_prefix"] = secret[:min(4, len(secret))]
			delete(key, "value")
		case req.Method == http.MethodGet && req.URL.Path == "/keys":
			result := make([]map[string]any, 0, len(keys))
			for _, key := range keys {
				result = append(result, key)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{"keys": result})
		case req.Method == http.MethodGet || req.Method == http.MethodDelete:
			keyID := pathSuffix(req.URL.Path, "/keys/")

			key, exists := keys[keyID]
			if !exists {
				w.WriteHeader(http.StatusNotFound)

				return
			}

			if req.Method == http.MethodDelete {
				delete(keys, keyID)
			}

			writeJSON(t, w, http.StatusOK, key)
		default:
			t.Errorf("unexpected key request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)

	config := func(setting string) string {
		return providerConfig(server.URL) + keyAutodeleteConfig(setting)
	}
	checks := func(id int64, autodelete bool) []statecheck.StateCheck {
		return []statecheck.StateCheck{
			statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("id"), knownvalue.Int64Exact(id)),
			statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("autodelete"), knownvalue.Bool(autodelete)),
			statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("value"), knownvalue.StringExact(fmt.Sprintf("generated-key-%d", id))),
			statecheck.ExpectKnownValue("data.typesense_key.test", tfjsonpath.New("autodelete"), knownvalue.Bool(autodelete)),
			statecheck.ExpectKnownOutputValue("listed", knownvalue.ObjectPartial(map[string]knownvalue.Check{"autodelete": knownvalue.Bool(autodelete)})),
		}
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("") + `import {
 to = typesense_key.test
 id = "99"
}`, ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("id"), knownvalue.Int64Exact(99)),
				statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("autodelete"), knownvalue.Bool(true)),
				statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("value"), knownvalue.Null()),
			}},
			{Config: config(""), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionNoop)},
			{Config: config("autodelete = false"), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace), ConfigStateChecks: checks(1, false)},
			{Config: config("autodelete = false"), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionNoop), ConfigStateChecks: checks(1, false)},
			{Config: config("autodelete = true"), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace), ConfigStateChecks: checks(2, true)},
			{Config: config(""), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionNoop), ConfigStateChecks: checks(2, true)},
			{ResourceName: "typesense_key.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"value"}},
			{Config: config("autodelete = false"), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace), ConfigStateChecks: checks(3, false)},
		},
	})

	mutex.Lock()
	defer mutex.Unlock()

	assert.Empty(t, keys)
	assert.Equal(t, 4, nextID)
}

func keyAutodeleteConfig(setting string) string {
	return fmt.Sprintf(`resource "typesense_key" "test" {
 description = "key autodelete lifecycle"
 actions = ["documents:search"]
 collections = ["posts"]
 %s
}
data "typesense_key" "test" { id = typesense_key.test.id }
data "typesense_keys" "test" { depends_on = [typesense_key.test] }
output "listed" { value = one([for key in data.typesense_keys.test.keys : key if key.id == typesense_key.test.id]) }
`, setting)
}

func keyPlanAction(action plancheck.ResourceActionType) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("typesense_key.test", action)}}
}

func TestAccKeyAutodelete(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: keyAutodeleteConfig(""), ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("autodelete"), knownvalue.Bool(false)),
			}},
			{Config: keyAutodeleteConfig("autodelete = true"), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace), ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("typesense_key.test", tfjsonpath.New("autodelete"), knownvalue.Bool(true)),
				statecheck.ExpectKnownValue("data.typesense_key.test", tfjsonpath.New("autodelete"), knownvalue.Bool(true)),
				statecheck.ExpectKnownOutputValue("listed", knownvalue.ObjectPartial(map[string]knownvalue.Check{"autodelete": knownvalue.Bool(true)})),
			}},
			{ResourceName: "typesense_key.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"value"}},
			{Config: keyAutodeleteConfig(""), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionNoop)},
			{Config: keyAutodeleteConfig("autodelete = false"), ConfigPlanChecks: keyPlanAction(plancheck.ResourceActionReplace)},
		},
	})
}
