package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/typesense/typesense-go/v3/typesense"
)

var _ function.Function = (*generateScopedSearchKeyFunction)(nil)

type generateScopedSearchKeyFunction struct{}

//nolint:ireturn
func NewGenerateScopedSearchKeyFunction() function.Function {
	return &generateScopedSearchKeyFunction{}
}

func (g *generateScopedSearchKeyFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "generate_scoped_search_key"
}

func (g *generateScopedSearchKeyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Generates a scoped search API key with embedded search parameters.",
		MarkdownDescription: "Generates a scoped search key locally using a parent search-only API key and embedded search parameters. No provider configuration or API request is needed.\n\n" +
			"The parent key must grant only `documents:search`. The function cannot verify the parent's permissions, validity or expiration, or whether Typesense accepts the embedded search parameters.\n\n" +
			"Use `sensitive(...)` and mark outputs containing the key sensitive. Sensitive values remain in Terraform state and saved plans. See [API key lifecycle](../guides/keys#scoped-search-keys) for expiration and revocation.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "search_key", MarkdownDescription: "Parent search-only API key, at least four bytes long. Scoped keys expose the first four bytes of the parent, including the entire parent when it is four bytes long."},
			function.DynamicParameter{Name: "params", MarkdownDescription: "Object or map of Typesense search parameters to embed, such as `filter_by`, `include_fields` and `expires_at`. Nested JSON-compatible values are supported. Explicit nulls are preserved. Unknown values defer generation until they are known."},
		},
		Return: function.StringReturn{},
	}
}

func (g *generateScopedSearchKeyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var (
		searchKey string
		params    types.Dynamic
	)

	resp.Error = req.Arguments.Get(ctx, &searchKey, &params)
	if resp.Error != nil {
		return
	}

	if len(searchKey) < keyPrefixLength {
		resp.Error = function.NewArgumentFuncError(0, "The parent search key must contain at least four bytes.")

		return
	}

	value, err := params.ToTerraformValue(ctx)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "Could not read search parameters.")

		return
	}

	parameters, err := scopedSearchParameters(value)
	if err != nil {
		message := "Search parameters must contain known JSON-compatible values."
		if errors.Is(err, errSearchParametersObject) {
			message = "Search parameters must be an object or map."
		}

		resp.Error = function.NewArgumentFuncError(1, message)

		return
	}

	client := typesense.Client{}

	key, err := client.Keys().GenerateScopedSearchKey(searchKey, parameters)
	if err != nil {
		resp.Error = function.NewFuncError("Could not generate the scoped search key.")

		return
	}

	resp.Error = resp.Result.Set(ctx, key)
}
