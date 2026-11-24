package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

func collectionExactFallback(fields types.List) (types.Object, bool) {
	if fields.IsNull() || fields.IsUnknown() {
		return types.Object{}, false
	}

	for _, value := range fields.Elements() {
		if value.IsNull() || value.IsUnknown() {
			continue
		}

		field := value.(types.Object)                     //nolint:forcetypeassert // The schema guarantees object elements.
		name := field.Attributes()["name"].(types.String) //nolint:forcetypeassert // The schema guarantees a string name.

		if !name.IsNull() && !name.IsUnknown() && name.ValueString() == ".*" {
			return field, true
		}
	}

	return types.Object{}, false
}
