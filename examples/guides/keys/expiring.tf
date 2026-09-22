resource "typesense_key" "temporary_search" {
  description = "Temporary search access"
  actions     = ["documents:search"]
  collections = ["posts"]
  expires_at  = 1893456000 # 2030-01-01T00:00:00Z
  autodelete  = true
}
