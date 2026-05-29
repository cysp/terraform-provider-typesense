resource "typesense_collection" "posts_v1" {
  name = "posts_v1"

  fields = [
    {
      name = "title"
      type = "string"
    },
  ]
}

resource "typesense_alias" "posts" {
  name            = "posts"
  collection_name = typesense_collection.posts_v1.name
}
