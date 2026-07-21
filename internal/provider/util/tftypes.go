package util

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	errUnsupportedType       = errors.New("unsupported type")
	errUnsupportedNumberType = errors.New("unsupported number type")
)

//nolint:cyclop
func UnmarshalTerraformValue(value tftypes.Value) (interface{}, error) {
	switch {
	case value.Type().Is(tftypes.Bool):
		var val *bool

		if err := value.As(&val); err != nil {
			return nil, fmt.Errorf("expected bool value: %w", err)
		}

		return val, nil
	case value.Type().Is(tftypes.Number):
		var val *big.Float

		if err := value.As(&val); err != nil {
			return nil, fmt.Errorf("expected number value: %w", err)
		}

		if int64Result, int64ResultAccuracy := val.Int64(); int64ResultAccuracy == big.Exact {
			return int64Result, nil
		}

		if float64Result, float64Accuracy := val.Float64(); float64Accuracy == big.Exact {
			return float64Result, nil
		}

		return nil, fmt.Errorf("%w: %v", errUnsupportedNumberType, val)
	case value.Type().Is(tftypes.String):
		var val *string

		if err := value.As(&val); err != nil {
			return nil, fmt.Errorf("expected string value: %w", err)
		}

		return val, nil
	case value.Type().Is(tftypes.List{}):
		fallthrough
	case value.Type().Is(tftypes.Set{}):
		fallthrough
	case value.Type().Is(tftypes.Tuple{}):
		var val []tftypes.Value

		if err := value.As(&val); err != nil {
			return nil, fmt.Errorf("expected list value: %w", err)
		}

		var result []interface{}

		for _, item := range val {
			itemValue, err := UnmarshalTerraformValue(item)
			if err != nil {
				return nil, err
			}

			result = append(result, itemValue)
		}

		return result, nil
	case value.Type().Is(tftypes.Map{}):
		fallthrough
	case value.Type().Is(tftypes.Object{}):
		var val map[string]tftypes.Value

		if err := value.As(&val); err != nil {
			return nil, fmt.Errorf("expected object value: %w", err)
		}

		result := make(map[string]interface{})

		for key, item := range val {
			itemValue, err := UnmarshalTerraformValue(item)
			if err != nil {
				return nil, err
			}

			result[key] = itemValue
		}

		return result, nil
	default:
		return nil, fmt.Errorf("%w: %v", errUnsupportedType, value)
	}
}
