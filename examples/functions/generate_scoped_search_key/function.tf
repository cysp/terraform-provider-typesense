resource "typesense_key" "search" {
  description = "Parent key for scoped product searches"
  actions     = ["documents:search"]
  collections = ["products"]
}

locals {
  scoped_search_key = sensitive(provider::typesense::generate_scoped_search_key(
    typesense_key.search.value,
    {
      filter_by      = "tenant_id:=customer_123"
      include_fields = "title,price"
      expires_at     = 2000000000
    }
  ))
}
