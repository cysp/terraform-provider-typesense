package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

// An auto declaration and its concrete expansion can share a name in the response.
func collectionFields(observed, previous []api.Field) []api.Field {
	used := make([]bool, len(observed))
	fields := make([]api.Field, 0, len(observed))

	for _, prior := range previous {
		index := -1

		for i, field := range observed {
			if !used[i] && field.Name == prior.Name {
				if index == -1 || field.Type == prior.Type {
					index = i
				}

				if field.Type == prior.Type {
					break
				}
			}
		}

		if index >= 0 {
			used[index] = true
			fields = append(fields, observed[index])
		}
	}

	for i, field := range observed {
		if !used[i] {
			fields = append(fields, field)
		}
	}

	return fields
}

func sameCollectionFields(observed, desired []api.Field) bool {
	if len(observed) != len(desired) {
		return false
	}

	fields := collectionFields(observed, desired)
	for i, want := range desired {
		if !sameCollectionField(fields[i], want) {
			return false
		}
	}

	return true
}

// Compare modeled attributes after applying Typesense defaults.
func sameCollectionField(a, b api.Field) bool {
	fields := CollectionFieldModelsFromAPI([]api.Field{normalizeCollectionField(a), normalizeCollectionField(b)})

	return reflect.DeepEqual(fields[0], fields[1])
}

func normalizeCollectionField(field api.Field) api.Field {
	if field.Facet == nil {
		field.Facet = new(false)
	}

	if field.Index == nil {
		field.Index = new(true)
	}

	if field.Infix == nil {
		field.Infix = new(false)
	}

	if field.Optional == nil {
		field.Optional = new(false)
	}

	if field.Sort == nil {
		field.Sort = new(false)
	}

	if field.Locale == nil {
		value := ""
		field.Locale = &value
	}

	return field
}

func (model *CollectionModel) readCollection(ctx context.Context, response *api.CollectionResponse) diag.Diagnostics {
	prior, diags := model.ToAPICollectionSchema(ctx)
	if model.Fields.IsNull() || model.Fields.IsUnknown() {
		diags = nil
		prior.Fields = nil
	}

	if diags.HasError() {
		return diags
	}

	responseCopy := *response
	responseCopy.Fields = collectionFields(response.Fields, prior.Fields)
	diags.Append(model.ReadFromResponse(ctx, &responseCopy)...)

	return diags
}

var errReindexUnstoredField = errors.New("cannot reindex field")

// A child's own store flag can be true even when Typesense discards the
// entire containing object before writing the document.
func validateStoredReindexSource(current *api.CollectionResponse, name string) error {
	for _, field := range current.Fields {
		if field.Store == nil || *field.Store {
			continue
		}

		containingObject := current.EnableNestedFields != nil && *current.EnableNestedFields &&
			(field.Type == "object" || field.Type == "object[]") && !isCollectionDynamicField(field) && strings.HasPrefix(name, field.Name+".")
		if field.Name == name || containingObject {
			return fmt.Errorf("%w %q: field %q has store=false, so Typesense cannot rebuild the index from stored documents. No schema alteration was sent. Create the destination schema through Typesense with store=false before loading documents, populate it from the original data, and switch the alias", errReindexUnstoredField, name, field.Name)
		}
	}

	return nil
}

func collectionFieldChanges(current *api.CollectionResponse, desired []api.Field) ([]api.Field, error) {
	previous := current.Fields

	removed, err := collectionFieldsToDrop(current, desired)
	if err != nil {
		return nil, err
	}

	// A drop applies to every observed definition with that name, including
	// fields implicitly dropped with a nested parent.
	for _, old := range previous {
		if old.Index != nil && !*old.Index {
			continue
		}

		sameName := func(field api.Field) bool { return field.Name == old.Name }
		if !slices.ContainsFunc(removed, sameName) || !slices.ContainsFunc(desired, sameName) {
			continue
		}

		err := validateStoredReindexSource(current, old.Name)
		if err != nil {
			return nil, err
		}
	}

	var changes []api.Field
	for _, old := range removed {
		changes = append(changes, api.Field{Name: old.Name, Drop: new(true)})
	}

	for _, want := range desired {
		dropped := slices.ContainsFunc(removed, func(field api.Field) bool { return field.Name == want.Name })
		if dropped || !slices.ContainsFunc(previous, func(field api.Field) bool { return field.Name == want.Name && field.Type == want.Type }) {
			changes = append(changes, want)
		}
	}

	return changes, nil
}

func collectionFieldsToDrop(current *api.CollectionResponse, desired []api.Field) ([]api.Field, error) {
	previous := current.Fields

	var removed []api.Field

	removedNames := make(map[string]bool)

	for _, old := range previous {
		index := slices.IndexFunc(desired, func(field api.Field) bool { return field.Name == old.Name && field.Type == old.Type })
		if (index < 0 || !sameCollectionField(old, desired[index])) && !removedNames[old.Name] {
			removed = append(removed, old)
			removedNames[old.Name] = true
		}
	}

	err := validateCollectionDynamicDrops(previous, removedNames)
	if err != nil {
		return nil, err
	}

	// Concrete nested fields have literal prefix cascades, independent of regexes.
	if current.EnableNestedFields != nil && *current.EnableNestedFields {
		for i := 0; i < len(removed); i++ {
			parent := removed[i]
			if isCollectionDynamicField(parent) || (!strings.Contains(parent.Name, ".") && !slices.Contains([]string{"object", "object[]"}, parent.Type)) {
				continue
			}

			for _, field := range previous {
				if !isCollectionDynamicField(field) && !removedNames[field.Name] && strings.HasPrefix(field.Name, parent.Name+".") {
					removed = append(removed, field)
					removedNames[field.Name] = true
				}
			}
		}
	}

	return removed, nil
}

var errCollectionDynamicDrop = errors.New("cannot safely remove or reindex dynamic field")

func validateCollectionDynamicDrops(previous []api.Field, removedNames map[string]bool) error {
	// Names are opaque. A dynamic drop may remove any retained concrete field;
	// reject that uncertainty instead of interpreting Typesense's regex dialect.
	for _, old := range previous {
		if !removedNames[old.Name] || old.Name == ".*" || !isCollectionDynamicField(old) {
			continue
		}

		for _, retained := range previous {
			if !isCollectionDynamicField(retained) && !removedNames[retained.Name] {
				return fmt.Errorf("%w %q while retaining concrete field %q. No schema alteration was sent. Remove the dynamic rule and existing concrete fields in a separate apply before adding the desired schema, or use a new collection and an alias cutover", errCollectionDynamicDrop, old.Name, retained.Name)
			}
		}
	}

	return nil
}

func isCollectionDynamicField(field api.Field) bool {
	return strings.Contains(field.Name, ".*") || slices.Contains([]string{"auto", "string*"}, field.Type)
}

func hasEmbeddingCredentials(field api.Field) bool {
	if field.Embed == nil {
		return false
	}

	// Typesense masks these model_config properties in collection reads.
	config := field.Embed.ModelConfig

	return config.ApiKey != nil || config.AccessToken != nil || config.RefreshToken != nil ||
		config.ClientId != nil || config.ClientSecret != nil || config.ProjectId != nil
}

func validateCollectionFieldAlteration(current *api.CollectionResponse, field api.Field) diag.Diagnostics {
	if field.Drop != nil && *field.Drop {
		return nil
	}

	if hasEmbeddingCredentials(field) {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Cannot reindex embedding field", fmt.Sprintf("Typesense masks embedding credentials when reading field %q. No schema alteration was sent. Apply this field change through Typesense with the original model configuration and credentials, then refresh and review a new Terraform plan.", field.Name))}
	}

	if current.EnableNestedFields == nil || !*current.EnableNestedFields {
		return nil
	}

	// Typesense restores original descendants when adding a nested parent.
	nested := field.Type == "object" || field.Type == "object[]" || (strings.Contains(field.Name, ".") && !strings.Contains(field.Name, ".*"))
	if !nested {
		return nil
	}

	for _, existing := range current.Fields {
		if strings.Contains(existing.Name, ".*") || slices.Contains([]string{"auto", "string*"}, existing.Type) {
			continue
		}

		if strings.HasPrefix(existing.Name, field.Name+".") {
			return diag.Diagnostics{diag.NewErrorDiagnostic("Overlapping collection field alteration", fmt.Sprintf("No schema alteration was sent. Field %q cannot be added or reindexed while nested field %q exists: Typesense can restore the old descendants or fail after changing the schema. Apply removal of the affected parent and children first, verify their absence, then apply the desired declarations. Remove rules or coordinate writes that could recreate these fields between applies. Alternatively, use a new collection and an alias cutover.", field.Name, existing.Name))}
		}
	}

	return nil
}
