# Terraform Provider for Typesense

Manage Typesense collections, collection aliases, and API keys with Terraform. Generate scoped search keys locally with [`generate_scoped_search_key`](docs/functions/generate_scoped_search_key.md).

Set `TYPESENSE_URL` to your cluster API endpoint and `TYPESENSE_API_KEY` to a key with the required permissions. For Typesense Cloud HA clusters, use the load-balanced endpoint.

```hcl
terraform {
  required_version = ">= 1.15, < 1.17"
  required_providers {
    typesense = {
      source = "cysp/typesense"
    }
  }
}

provider "typesense" {}

resource "typesense_collection" "posts" {
  name = "posts"
  fields = [
    { name = "title", type = "string" },
  ]
}

resource "typesense_alias" "posts" {
  name            = "current_posts"
  collection_name = typesense_collection.posts.name
}
```

CI tests Terraform **1.15 and 1.16** with Typesense **30.2**.

Collection field changes alter existing collections in place. Terraform manages every observed field, including fields inferred from document ingestion; fields absent from configuration are planned for index removal. Review the [collection lifecycle guide](docs/guides/collection-lifecycle.md) before changing a production schema, and the [upgrade guide](docs/guides/upgrading.md) when upgrading an existing configuration.

See the [provider documentation](docs/index.md), [examples](examples), and [development instructions](DEVELOPMENT.md). The [API key lifecycle guide](docs/guides/keys.md) covers metadata lookup, expiration and rotation. The API key `value` attribute is sensitive in Terraform output but remains in Terraform state. The server-provided `value_prefix` is not sensitive and can contain an entire short key; see the key lifecycle guide for details.
