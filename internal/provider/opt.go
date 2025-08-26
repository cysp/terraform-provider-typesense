package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type gettableOpt[T any] interface {
	Get() (T, bool)
}

func getOptInt64Value[T gettableOpt[int64]](ov T) types.Int64 {
	val, ok := ov.Get()
	if !ok {
		return types.Int64Null()
	}

	return types.Int64Value(val)
}

type settableOpt[T any] interface {
	SetTo(v T)
	Reset()
}

func setOptInt64FromValue[T settableOpt[int64]](ov T, value types.Int64) {
	if value.IsNull() || value.IsUnknown() {
		ov.Reset()
	} else {
		ov.SetTo(value.ValueInt64())
	}
}

func setOptStringFromValue[T settableOpt[string]](ov T, value types.String) {
	if value.IsNull() || value.IsUnknown() {
		ov.Reset()
	} else {
		ov.SetTo(value.ValueString())
	}
}
