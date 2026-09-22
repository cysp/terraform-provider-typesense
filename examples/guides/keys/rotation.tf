# Add the next generation, migrate consumers, then remove the old generation.
resource "typesense_key" "search" {
  for_each = toset(["generation_1", "generation_2"])

  description = "Search access: ${each.key}"
  actions     = ["documents:search"]
  collections = ["posts"]
}
