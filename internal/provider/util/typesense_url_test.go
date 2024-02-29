package util_test

import (
	"os"
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/stretchr/testify/assert"
)

func TestTypesenseUrlFromEnv(t *testing.T) {
	if os.Getenv("TF_ACC") != "" {
		return
	}

	tests := map[string]struct {
		env         map[string]string
		expectedURL string
	}{
		"absent": {
			env: map[string]string{},
		},
		"url: empty string": {
			env: map[string]string{"TYPESENSE_URL": ""},
		},
		"url: valid": {
			env:         map[string]string{"TYPESENSE_URL": "http://example.com"},
			expectedURL: "http://example.com",
		},
		"url: invalid": {
			env:         map[string]string{"TYPESENSE_URL": "invalid.url"},
			expectedURL: "invalid.url",
		},
		"components: protocol": {
			env: map[string]string{"TYPESENSE_PROTOCOL": "http"},
		},
		"components: protocol,host": {
			env:         map[string]string{"TYPESENSE_PROTOCOL": "http", "TYPESENSE_HOST": "typesense.example.org"},
			expectedURL: "http://typesense.example.org",
		},
		"components: protocol,port": {
			env: map[string]string{"TYPESENSE_PROTOCOL": "http", "TYPESENSE_PORT": "8108"},
		},
		"components: protocol,host,port": {
			env:         map[string]string{"TYPESENSE_PROTOCOL": "http", "TYPESENSE_HOST": "typesense.example.org", "TYPESENSE_PORT": "8108"},
			expectedURL: "http://typesense.example.org:8108",
		},
		"components: host": {
			env:         map[string]string{"TYPESENSE_HOST": "typesense.example.org"},
			expectedURL: "http://typesense.example.org",
		},
		"components: empty host": {
			env: map[string]string{"TYPESENSE_HOST": ""},
		},
		"components: host,port": {
			env:         map[string]string{"TYPESENSE_HOST": "typesense.example.org", "TYPESENSE_PORT": "8108"},
			expectedURL: "http://typesense.example.org:8108",
		},
		"components: port": {
			env: map[string]string{"TYPESENSE_PORT": "8108"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for key, value := range test.env {
				t.Setenv(key, value)
			}

			actualURL, actualFound := util.TypesenseURLFromEnv()

			assert.EqualValues(t, test.expectedURL, actualURL)
			assert.EqualValues(t, test.expectedURL != "", actualFound)
		})
	}
}
