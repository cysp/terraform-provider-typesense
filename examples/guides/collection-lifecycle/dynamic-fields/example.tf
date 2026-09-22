resource "typesense_collection" "posts" {
  name = "posts"
  fields = [
    { name = "title", type = "string", facet = true },
    { name = ".*", type = "auto", optional = true },
  ]
}
