package provider

import (
	"encoding/json"
	"testing"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

// The serialized parameters become part of the scoped key. Preserve the JSON
// value, including integer precision, explicit nulls and escaped strings.
func FuzzScopedSearchParametersRoundTrip(f *testing.F) {
	f.Add("user:\"example\"", int64(9007199254740993))
	f.Add("", int64(-1))

	f.Fuzz(func(t *testing.T, filter string, expiresAt int64) {
		if !utf8.ValidString(filter) {
			t.Skip("Terraform strings contain Unicode characters")
		}

		input, err := json.Marshal(map[string]any{
			"expires_at": expiresAt,
			"nested":     []any{filter, nil},
		})
		require.NoError(t, err)

		nested := tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, filter), tftypes.NewValue(tftypes.String, nil),
		})
		value := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"expires_at": tftypes.Number, "nested": nested.Type(),
		}}, map[string]tftypes.Value{
			"expires_at": tftypes.NewValue(tftypes.Number, expiresAt), "nested": nested,
		})
		params, err := scopedSearchParameters(value)
		require.NoError(t, err)
		output, err := json.Marshal(params)
		require.NoError(t, err)
		require.Equal(t, string(input), string(output))
	})
}
