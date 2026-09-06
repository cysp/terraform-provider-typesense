package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/typesense/typesense-go/typesense"
)

var _ function.Function = &generateScopedSearchKeyFunction{}

type generateScopedSearchKeyFunction struct{}

//nolint:ireturn
func NewGenerateScopedSearchKeyFunction() function.Function {
	return &generateScopedSearchKeyFunction{}
}

func (g *generateScopedSearchKeyFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "generate_scoped_search_key"
}

// Definition implements function.Function.
func (g *generateScopedSearchKeyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Parameters: []function.Parameter{
			function.StringParameter{
				Name: "search_key",
			},
			function.DynamicParameter{
				Name: "params",
			},
		},
		Return: function.StringReturn{},
	}
}

func (g *generateScopedSearchKeyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var (
		searchKeyArgument string
		paramsArgument    types.Dynamic
	)

	funcError := req.Arguments.Get(ctx, &searchKeyArgument, &paramsArgument)
	if funcError != nil {
		resp.Error = funcError

		return
	}

	client := typesense.Client{}

	paramsArgumentValue, err := paramsArgument.ToTerraformValue(ctx)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())

		return
	}

	if !paramsArgumentValue.IsFullyKnown() {
		resp.Error = function.NewArgumentFuncError(1, "params argument must be fully known")

		return
	}

	paramsInterface, err := util.UnmarshalTerraformValue(paramsArgumentValue)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())

		return
	}

	params, ok := paramsInterface.(map[string]interface{})
	if !ok {
		resp.Error = function.NewFuncError("params argument must be a map")

		return
	}

	scopedSearchKey, err := client.Keys().GenerateScopedSearchKey(searchKeyArgument, params)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())

		return
	}

	resp.Error = resp.Result.Set(ctx, scopedSearchKey)
}
