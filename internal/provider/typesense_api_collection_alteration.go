package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

//nolint:tagliatelle // Typesense field property names.
type alterationField struct {
	api.Field

	AsyncReference *bool           `json:"async_reference,omitempty"`
	CascadeDelete  *bool           `json:"cascade_delete,omitempty"`
	HNSWParams     json.RawMessage `json:"hnsw_params,omitempty"`
	TruncateLen    json.RawMessage `json:"truncate_len,omitempty"`
}

type observedField struct {
	*api.Field

	raw json.RawMessage
}

type observedCollection struct {
	schema       api.CollectionResponse
	fieldsByName map[string][]observedField
}

// Decode the collection envelope once. Each field's typed identity and original
// definition come from the same JSON value, including when names are repeated.
func decodeObservedCollection(body []byte) (*observedCollection, error) {
	var response struct {
		api.CollectionResponse

		Fields []json.RawMessage `json:"fields"`
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("decode collection: %w", err)
	}

	observed := &observedCollection{schema: response.CollectionResponse, fieldsByName: make(map[string][]observedField)}
	if response.Fields != nil {
		observed.schema.Fields = make([]api.Field, len(response.Fields))
	}

	for i, raw := range response.Fields {
		field := &observed.schema.Fields[i]

		err := json.Unmarshal(raw, field)
		if err != nil {
			return nil, fmt.Errorf("decode collection field: %w", err)
		}

		observed.fieldsByName[field.Name] = append(observed.fieldsByName[field.Name], observedField{Field: field, raw: raw})
	}

	return observed, nil
}

func (c *observedCollection) encodeAlteration(changes []api.Field) ([]byte, diag.Diagnostics) {
	var patch struct {
		Fields []alterationField `json:"fields"`
	}

	for _, change := range changes {
		field, err := c.fieldForAlteration(change)
		if err != nil {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Cannot safely rebuild collection field", err.Error())}
		}

		if diags := validateCollectionFieldAlteration(&c.schema, field.Field); diags.HasError() {
			return nil, diags
		}

		patch.Fields = append(patch.Fields, field)
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Cannot encode collection alteration", err.Error())}
	}

	return body, nil
}

var errAmbiguousFieldSource = errors.New("ambiguous source for field")

func (c *observedCollection) fieldForAlteration(change api.Field) (alterationField, error) {
	candidates := c.fieldsByName[change.Name]
	if (change.Drop != nil && *change.Drop) || len(candidates) == 0 {
		return alterationField{Field: change}, nil
	}

	if len(candidates) > 1 {
		var exact []observedField

		for _, field := range candidates {
			if field.Type == change.Type {
				exact = append(exact, field)
			}
		}

		if len(exact) > 0 {
			candidates = exact
		}
	}

	if len(candidates) == 1 {
		return rebuildCollectionField(candidates[0].raw, change)
	}

	types := make([]string, 0, len(candidates))
	for _, field := range candidates {
		types = append(types, field.Type)
	}

	slices.Sort(types)

	return alterationField{}, fmt.Errorf("%w %q of type %q: observed types %s. No alteration was sent. Resolve the field definitions in Typesense, then refresh and review a new plan", errAmbiguousFieldSource, change.Name, change.Type, strings.Join(types, ", "))
}

func rebuildCollectionField(raw json.RawMessage, want api.Field) (alterationField, error) {
	var field alterationField

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&field)
	if err != nil {
		return alterationField{}, fmt.Errorf("cannot safely reindex field %q: %w. No alteration was sent. Use a provider version that supports this field definition or apply the field change through Typesense, then refresh and review a new plan", want.Name, err)
	}

	field.Type, field.Facet, field.Index, field.Infix, field.Locale = want.Type, want.Facet, want.Index, want.Infix, want.Locale

	field.NumDim, field.Optional, field.Reference, field.Sort = want.NumDim, want.Optional, want.Reference, want.Sort
	if field.Reference == nil || *field.Reference == "" {
		field.AsyncReference, field.CascadeDelete = nil, nil
	}

	if field.NumDim == nil && field.Embed == nil {
		field.VecDist, field.HNSWParams = nil, nil
	}

	return field, nil
}
