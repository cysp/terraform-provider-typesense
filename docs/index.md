---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "typesense Provider"
subcategory: ""
description: |-
  
---

# typesense Provider



## Example Usage

```terraform
provider "typesense" {
  url = "https://typesense.example.org:8108"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `api_key` (String) Typesense Admin API Key. Alternatively, can be configured using the `TYPESENSE_API_KEY` environment variable.
- `url` (String) Typesense API URL. Alternatively, can be configured using the `TYPESENSE_URL` environment variable. Alternatively alternatively, can be configured using the `TYPESENSE_PROTOCOL`, `TYPESENSE_HOST` and `TYPESENSE_PORT` environment variables.
