resource "typesense_collection" "posts" {
  name = "posts"
  fields = [
    { name = ".*", type = "auto", optional = true },
  ]

  lifecycle {
    ignore_changes = [fields]
  }
}
