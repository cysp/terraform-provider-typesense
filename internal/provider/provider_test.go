package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"typesense": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func providerConfigDynamicValue(config map[string]interface{}) (tfprotov6.DynamicValue, error) {
	providerConfigTypes := map[string]tftypes.Type{
		"url":     tftypes.String,
		"api_key": tftypes.String,
	}
	providerConfigObjectType := tftypes.Object{AttributeTypes: providerConfigTypes}

	providerConfigObjectValue := tftypes.NewValue(providerConfigObjectType, map[string]tftypes.Value{
		"url":     tftypes.NewValue(tftypes.String, config["url"]),
		"api_key": tftypes.NewValue(tftypes.String, config["api_key"]),
	})

	value, err := tfprotov6.NewDynamicValue(providerConfigObjectType, providerConfigObjectValue)
	if err != nil {
		err = fmt.Errorf("failed to create dynamic value: %w", err)
	}

	return value, err
}

func TestProtocol6ProviderServerSchemaVersion(t *testing.T) {
	t.Parallel()

	providerServer, err := testAccProtoV6ProviderFactories["typesense"]()
	require.NotNil(t, providerServer)
	require.NoError(t, err)

	resp, err := providerServer.GetProviderSchema(t.Context(), &tfprotov6.GetProviderSchemaRequest{})
	require.NotNil(t, resp.Provider)
	require.NoError(t, err)
	assert.Empty(t, resp.Diagnostics)

	assert.EqualValues(t, 0, resp.Provider.Version)
}

func TestProtocol6ProviderServerConfigure(t *testing.T) {
	if os.Getenv("TF_ACC") != "" {
		return
	}

	tests := map[string]struct {
		config          map[string]interface{}
		env             map[string]string
		expectedSuccess bool
	}{
		"config: url": {
			config: map[string]interface{}{
				"url": "http://localhost:8108",
			},
			expectedSuccess: false,
		},
		"config: api_key": {
			config: map[string]interface{}{
				"api_key": "12345",
			},
			expectedSuccess: false,
		},
		"config: url,api_key": {
			config: map[string]interface{}{
				"url":     "http://localhost:8108",
				"api_key": "12345",
			},
			expectedSuccess: true,
		},
		"env: url": {
			env: map[string]string{
				"TYPESENSE_URL": "http://localhost:8108",
			},
			expectedSuccess: false,
		},
		"env: url,api_key": {
			env: map[string]string{
				"TYPESENSE_URL":     "http://localhost:8108",
				"TYPESENSE_API_KEY": "12345",
			},
			expectedSuccess: true,
		},
		"config: url env: api_key": {
			config: map[string]interface{}{
				"url": "http://localhost:8108",
			},
			env: map[string]string{
				"TYPESENSE_API_KEY": "12345",
			},
			expectedSuccess: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for key, value := range test.env {
				t.Setenv(key, value)
			}

			providerServer, err := testAccProtoV6ProviderFactories["typesense"]()
			require.NotNil(t, providerServer)
			require.NoError(t, err)

			providerConfigValue, err := providerConfigDynamicValue(test.config)
			require.NotNil(t, providerConfigValue)
			require.NoError(t, err)

			resp, err := providerServer.ConfigureProvider(t.Context(), &tfprotov6.ConfigureProviderRequest{
				Config: &providerConfigValue,
			})
			require.NotNil(t, resp)
			require.NoError(t, err)

			if test.expectedSuccess {
				assert.Empty(t, resp.Diagnostics)
			} else {
				assert.NotEmpty(t, resp.Diagnostics)
			}
		})
	}
}
