variable "search_include_fields" {
  type    = string
  default = null
}

locals {
  scoped_search_params = merge(
    { filter_by = "tenant_id:=customer_123" },
    var.search_include_fields == null ? {} : { include_fields = var.search_include_fields },
  )
}
