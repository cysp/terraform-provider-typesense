---
page_title: "Collection lifecycle"
subcategory: ""
description: |-
  Manage collection fields, plan schema changes, and recover interrupted operations.
---

# Collection lifecycle

## Schema changes

Supported changes to `typesense_collection.fields` update the existing collection in place. Reordering fields alone does not alter the schema. Changing the collection name or explicitly changing `default_sorting_field`, `enable_nested_fields`, `symbols_to_index`, or `token_separators` requires replacement.

Removing a field removes its schema and index entry. Values already stored in documents remain, but are no longer searchable through that field. Existing documents must satisfy the resulting schema: for example, adding a required field fails if existing documents do not contain it. See the [Typesense alteration constraints](https://typesense.org/docs/30.2/api/collections.html#update-or-alter-a-collection).

Alterations can block writes while reindexing. When you need a separate cutover, create a collection with a new name, populate it, then update a `typesense_alias` to point to it. Populate and validate the documents using your application or migration tooling.

Typesense 29.1 rejects adding or modifying reference fields on existing collections, including changes to their other settings. Typesense 30.2 supports these alterations. On 29.1, use a new collection or upgrade the server for these changes. API rejection does not cause automatic collection replacement.

## Field defaults and upgrades

The provider gives omitted field options explicit effective values in the Terraform plan. Removing an option from configuration has the same result as never setting it: the next plan compares the Typesense field with that default. Refresh records the server's observed schema and never alters it.

For `sort`, the [Typesense 29.1](https://github.com/typesense/typesense/blob/v29.1/src/field.cpp#L29-L42) and [30.2](https://github.com/typesense/typesense/blob/v30.2/src/field.cpp#L27-L40) implementations default to `true` for scalar `int32`, `int64`, `float`, and `bool`, and for `geopoint`, `geopoint[]`, and `geopolygon`; other types, including arrays and `auto`, default to `false`. Geo fields cannot use `sort = false`. The former provider default was `false` for every field. After upgrading, an existing numeric, boolean, or geo field whose omitted `sort` was recorded as `false` can therefore show an in-place field alteration. Terraform state cannot tell whether that old value came from an explicit setting or the former default. Set `sort = false` explicitly on a non-geo field to keep it, or review and apply the reindex plan to adopt the Typesense default.

The newly managed options default to `store = true`, `range_index = false`, `stem = false`, an empty `stem_dictionary`, and empty field `token_separators` and `symbols_to_index` lists. A nonempty `stem_dictionary` makes omitted `stem` default to `true`; explicit `stem = false` conflicts with it. A `float[]` field with `num_dim` defaults `vec_dist` to `cosine`; `vec_dist` is unset on nonvector fields. Empty field tokenization lists inherit the collection-level settings when present; nonempty lists override them for that field. `range_index = true` requires a numerical field, and stemming requires a string or string array field. Typesense ignores these options on the exact `.*` fallback field, so the provider rejects nondefault values there before sending a schema change.

These defaults apply to imported collections and existing state as well as new fields. If Typesense reports a nondefault option that is absent from configuration, the next plan may drop and readd the field index to reset it. Declare the observed value explicitly to retain it. Review upgrade and import plans before applying, especially when existing fields have nondefault `store` settings: removing `store = false` or leaving it undeclared makes subsequent document writes store that field value. Index alterations can block writes.

### Storage and restart behavior

Typesense removes `store = false` field values before saving new documents. Changing a field from `store = true` to `false` does not erase values already stored; changing it back to `true` cannot recover values omitted from later writes. Reingest from your source if uniform stored data is required.

In direct snapshot and restart tests on Typesense 29.1 and 30.2, a required `store = false` field caused documents to disappear from search after restart. An optional `store = false` field retained the documents but stopped matching searches on that field. The [upstream required-field issue](https://github.com/typesense/typesense/issues/3022) documents the former behavior. The provider allows these configurations and reports the schema Typesense returns; a matching schema does not establish search-index durability. Validate searches after restart before relying on `store = false` for indexed fields.

### Typesense 29.1 vector distance persistence

Typesense 29.1 can report `vec_dist = "ip"` immediately after creating or altering a vector field, then report `cosine` after snapshot and restart. Its [restart loader](https://github.com/typesense/typesense/blob/v29.1/src/collection_manager.cpp#L110-L116) parses the option name instead of its value. Typesense 30.2 retained `ip` in direct tests. Refresh records the actual server value, so a 29.1 restart can produce another plan to restore `ip`; applying that plan does not guarantee that `ip` survives the next restart.

## Field ownership

Terraform manages the complete field schema returned by Typesense. Each configured field name must be unique. Every observed field belongs in `fields` if its index is to be retained. Fields absent from configuration and differences in configured field settings are reported as drift and reconciled on apply. Removing a field from configuration removes its index, not its stored document values.

This includes concrete fields inferred from `auto`, `string*`, regex rules such as `meta_.*`, and nested object fields. A dynamic rule does not exempt its inferred fields from Terraform management. Document ingestion can therefore produce new drift even when the Terraform configuration has not changed.

```terraform
resource "typesense_collection" "posts" {
  name = "posts"
  fields = [
    { name = "title", type = "string", facet = true },
    { name = ".*", type = "auto", optional = true },
  ]
}
```

In this example, a document containing an additional `score` field can cause Typesense to infer a concrete field. The next refresh includes `score` in Terraform state, and the next plan proposes removing its index unless its declaration is added to `fields`. Further document writes can recreate that index and produce another removal plan.

To retain an observed field, declare it with its observed settings before applying. Declaring matching settings does not itself require reindexing. Supported setting changes alter the existing field. Review the complete plan and representative searches when adopting or removing fields.

A named `auto` or `string*` declaration can produce a concrete field with the same name. The resulting schema cannot be represented by unique configured names. Use concrete declarations for exact management, or ignore the entire `fields` attribute when Typesense should control inference. Migrating an existing dynamic schema may require the separate applies described below.

Field names and regex rules are passed to Typesense as opaque strings. The provider does not evaluate whether a name matches a rule. Preservation of settings outside the resource schema is limited to attributes the provider's Typesense client can read and send.

### Letting another system manage fields

If Typesense or another system should manage the field schema after creation, ignore the entire `fields` attribute:

```terraform
resource "typesense_collection" "posts" {
  name = "posts"
  fields = [
    { name = ".*", type = "auto", optional = true },
  ]

  lifecycle {
    ignore_changes = [fields]
  }
}
```

Terraform still uses `fields` when creating the collection. Afterwards, `ignore_changes = [fields]` suppresses updates caused by both remote field drift and changes to the configured field list. Remove it only after reconciling configuration with the fields you intend to retain and reviewing the resulting plan.

Terraform's [`ignore_changes`](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#ignore_changes) does not select fields by name or regex. Individual list indexes refer to positions, not field identities; using them for a changing schema can ignore the wrong field. Ignoring the entire attribute is the broad opt-out from field management.

## Dynamic rule alterations

Removing or changing a dynamic rule can cause Typesense to remove concrete fields that match it, including explicitly configured fields. The provider rejects dropping or reindexing an opaque dynamic rule when the same alteration leaves any existing concrete field untouched, before sending the alteration. The `.*` fallback is exempt from this guard because removing it does not natively remove its concrete fields; the provider still removes any fields absent from configuration.

Use separate applies to remove the affected schema and then add the desired declarations, coordinating writers so fields cannot be recreated between stages. Alternatively, use a new collection and an alias cutover. Review both stages: removing fields makes their indexes unavailable until they are restored.

An alteration can also infer additional fields from stored documents. The provider requires the resulting field schema to match the plan exactly. If Typesense keeps returning additional fields, verification cannot succeed and the update can reach its timeout. The provider does not send another alteration to remove those fields automatically. Refresh, declare the fields you intend to retain, and review a new plan before applying again.

## Nested parent fields

Typesense can restore existing descendant fields when a parent is added or reindexed. Combining parent and child changes can therefore fail after partially changing the schema. The provider rejects adding or readding a parent with existing concrete descendants before sending the alteration. The diagnostic identifies the parent and a conflicting child. This includes object fields and non-regex dotted names, such as a parent `person` with a descendant `person.name`.

Unchanged parents, changes to children without descendants, and new parents without existing descendants remain supported. Removing a parent without readding it is also supported, including retaining explicitly declared children whose additions do not overlap other descendants.

To migrate an affected schema:

1. Coordinate document writers so inferred fields cannot reappear during the migration.
2. Remove the affected parent, child declarations and any rules that would recreate them. Apply and verify their absence from the schema.
3. Add the desired declarations in a separate apply, then verify the schema and representative searches.

Stored document values remain, but the affected indexes are unavailable between applies. Use a new collection and alias cutover if that interruption is unacceptable. Coordinate external schema changes as well: the schema can change between the provider's check and its alteration request.

Each stage is a separate practitioner-controlled apply. The provider does not automatically stage changes or roll them back after an error.

## Changes after planning

Apply verifies that the observed managed field schema matches either the schema recorded when planning or the planned result. If it already matches the planned result, the provider sends no alteration. If it matches neither, apply stops before altering the schema. Run a new plan with refresh enabled and review its changes before applying again.

This check includes fields inferred after planning, so document ingestion can make a saved plan unusable. It compares the field properties represented in Terraform; supported API properties outside the resource schema are preserved during alteration. The check and alteration are separate requests, so coordinate external schema writers as well.

Changing only `timeouts` does not read or alter the remote schema during the update. Normal Terraform refresh still reads the collection.

## Timeouts

Collection updates default to thirty minutes; create, read and delete operations default to five minutes. Override these limits with `timeouts`:

```terraform
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
```

The update deadline includes waiting for another alteration using the same provider configuration, reading the schema, sending the alteration and verifying its result. Terraform cancellation or an earlier deadline takes precedence. Typesense permits one cluster-wide alteration at a time, so coordinate changes from other processes and provider configurations.

## Recovery

A timeout, connection failure or server error can occur after Typesense accepts an alteration. The provider sends each mutation once and does not follow redirects that could repeat it. On an uncertain result, it reports an error and retains the resource identity.

Before applying again:

1. Wait for alteration activity to finish. Use `GET /operations/schema_changes` to inspect it. A healthy server response alone does not establish that an interrupted alteration has finished after a restart.
2. Verify the schema and representative searches.
3. Run a new Terraform plan with refresh enabled and review it before applying. The new plan records the current schema; apply verifies it before sending the planned alteration.

On Typesense 29.1 and 30.2, schema-change monitoring requires `operations/schema_changes:list`. This can be granted to a separate monitoring key; it is not required by the provider. The endpoint reports cluster-wide status regardless of the key's collection restrictions.

## High availability

For Typesense Cloud HA clusters, use the managed load-balanced endpoint. For self-hosted clusters, configure an API endpoint that routes to healthy nodes. The provider uses one URL and does not discover leaders or fail over between node addresses.

Followers forward writes to the leader but serve reads locally. After a successful alteration, the provider waits for consecutive reads to match the exact planned field schema within the update deadline. It does not repeat the alteration while waiting.

A replica can also briefly report that an existing resource is missing. The provider rechecks not-found responses over a five-second confirmation period and requires a final completed not-found read before removing the resource from state. Reads remain subject to the operation deadline. Cancellation, authentication failures and other read errors retain state.

These checks accommodate brief replication lag; they do not establish that every replica is current. If a plan still shows unexpected changes immediately after an operation, allow replication to settle and review a new plan before applying.
