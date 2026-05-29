resource "typesense_collection" "posts" {
  name = "posts"

  fields = [
    {
      name = "title"
      type = "string"
    },
    {
      name  = "published_at"
      type  = "int64"
      facet = true
      sort  = true
    },
  ]

  default_sorting_field = "published_at"
}
