package provider_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func providerConfig(url string) string {
	return `
	provider "typesense" {
		url = "` + url + `"
		api_key = "test"
	}
	`
}

func pathSuffix(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

func writeJSON(t *testing.T, w http.ResponseWriter, statusCode int, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	assert.NoError(t, json.NewEncoder(w).Encode(value))
}
