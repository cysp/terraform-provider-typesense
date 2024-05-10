resource "typesense_key" "this" {
  actions     = ["documents:search"]
  collections = ["posts"]
  description = "Key for searching posts."
}
