package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	errUnsupportedSearchParameter = errors.New("search parameter is not a known JSON-compatible value")
	errSearchParametersObject     = errors.New("search parameters must be an object or map")
)

func scopedSearchParameters(value tftypes.Value) (map[string]any, error) {
	if !value.Type().Is(tftypes.Object{}) && !value.Type().Is(tftypes.Map{}) {
		return nil, errSearchParametersObject
	}

	if !value.IsKnown() {
		return nil, errUnsupportedSearchParameter
	}

	if value.IsNull() {
		return nil, errSearchParametersObject
	}

	var values map[string]tftypes.Value

	err := value.As(&values)
	if err != nil {
		return nil, fmt.Errorf("read object search parameter: %w", err)
	}

	parameters := make(map[string]any, len(values))
	for name, child := range values {
		converted, err := scopedSearchParameterValue(child)
		if err != nil {
			return nil, err
		}

		parameters[name] = converted
	}

	return parameters, nil
}

func scopedSearchParameterValue(value tftypes.Value) (any, error) {
	if !value.IsKnown() {
		return nil, errUnsupportedSearchParameter
	}

	if value.IsNull() {
		return nil, nil //nolint:nilnil // Terraform null is a valid JSON null value.
	}

	switch {
	case value.Type().Is(tftypes.String):
		var result string

		err := value.As(&result)
		if err != nil {
			return nil, fmt.Errorf("read string search parameter: %w", err)
		}

		return result, nil
	case value.Type().Is(tftypes.Bool):
		var result bool

		err := value.As(&result)
		if err != nil {
			return nil, fmt.Errorf("read boolean search parameter: %w", err)
		}

		return result, nil
	case value.Type().Is(tftypes.Number):
		var result big.Float

		err := value.As(&result)
		if err != nil {
			return nil, fmt.Errorf("read numeric search parameter: %w", err)
		}

		if result.IsInf() {
			return nil, errUnsupportedSearchParameter
		}
		// json.Number avoids float64 rounding and big.Float's JSON string encoding.
		return json.Number(result.Text('f', -1)), nil
	case value.Type().Is(tftypes.Object{}), value.Type().Is(tftypes.Map{}):
		return scopedSearchParameters(value)
	case value.Type().Is(tftypes.List{}), value.Type().Is(tftypes.Set{}), value.Type().Is(tftypes.Tuple{}):
		var values []tftypes.Value

		err := value.As(&values)
		if err != nil {
			return nil, fmt.Errorf("read sequence search parameter: %w", err)
		}

		result := make([]any, len(values))
		for index, child := range values {
			converted, err := scopedSearchParameterValue(child)
			if err != nil {
				return nil, err
			}

			result[index] = converted
		}

		return result, nil
	default:
		return nil, errUnsupportedSearchParameter
	}
}
