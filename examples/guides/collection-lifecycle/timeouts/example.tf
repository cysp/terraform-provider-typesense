resource "typesense_collection" "posts" {
  name = "posts"
  fields = [
    { name = "title", type = "string" },
  ]

  timeouts = {
    create = "5m"
    read   = "5m"
    update = "45m"
    delete = "5m"
  }
}
