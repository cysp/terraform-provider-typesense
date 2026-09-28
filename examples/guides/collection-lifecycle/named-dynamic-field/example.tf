resource "typesense_collection" "events" {
  name = "events"
  fields = [
    { name = "score", type = "auto" },
    { name = "score", type = "int64", optional = true },
  ]
}
