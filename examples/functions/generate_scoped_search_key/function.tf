output "scoped_search_key" {
  value = provider::typesense::generate_scoped_search_key(typesense_key.this.value, {
    "scope" : "foo",
    "something" : "123",
    "another" : ["1", 2],
    "yetanother" : {
      "a" : 1,
      "b" : [2, "3"],
    },
  })
}
