import {
  to       = typesense_collection.posts
  identity = { name = "posts" }
}

import {
  to       = typesense_alias.current
  identity = { name = "current_posts" }
}

import {
  to       = typesense_key.search
  identity = { id = 123 }
}
