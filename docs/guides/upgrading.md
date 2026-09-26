---
page_title: "Upgrading from provider 0.0.6"
subcategory: ""
description: |-
  Configuration and lifecycle changes when upgrading from provider 0.0.6.
---

# Upgrading from provider 0.0.6

This guide describes the changes from provider 0.0.6. Back up Terraform state before upgrading, and review the first plan before applying it. A provider downgrade does not reverse schema or index changes already applied to Typesense.

## Key data source

The `typesense_key` data source retrieves metadata by `id`. Remove configured
`expires_at` and `value` arguments. `expires_at` remains available as a read-only
result. The `value` attribute has been removed. In provider 0.0.6 it echoed the
configured secret; the key metadata endpoint does not return a `value` field.

Pass an existing secret directly from its original source instead of through
this data source. The `typesense_key` **resource** still returns a key's secret
when creating it and preserves it in state across refresh. Importing a key
does not populate `value`. The server-provided `value_prefix` is used unchanged
and can contain an entire short key; see [API key lifecycle](keys).

## Collection lifecycle

Field changes now update the collection in place instead of planning replacement. Review the [collection lifecycle guide](collection-lifecycle) for field ownership, write blocking, dynamic-rule and nested-parent restrictions, and recovery after interrupted operations. Existing collection field attribute names and list types are retained. Collection names and explicitly changed immutable collection settings still require replacement.

The provider manages every field returned by Typesense, including concrete fields inferred from document ingestion. On refresh, fields missing from configuration appear as drift. An unchanged configuration can therefore plan an in-place update that removes existing indexes, whether or not provider 0.0.6 previously recorded those fields in state.

To retain those indexes, declare every field you intend to keep with its observed settings before applying. Dynamic rules do not exempt inferred fields from this requirement. If another system should manage fields after creation, `ignore_changes = [fields]` in the resource lifecycle opts out of all field updates, including configured changes. See the lifecycle guide for that tradeoff. Alternatively, migrate through a new collection and alias cutover.

A named `auto` or `string*` declaration can share its name with one concrete inferred field. Declare both rows to retain and manage the pair; other duplicate field names remain invalid.

If you apply the removals, stored document values survive, but the affected fields stop being searchable until they are indexed again. Remaining dynamic rules can rediscover them on subsequent document writes, producing further drift; existing documents are not automatically rewritten.

Omitted `optional` now resolves to `true` for dynamic declarations, matching the Typesense default. An older dynamic field recorded as `optional = false` can therefore plan an alteration; declare `false` explicitly only where Typesense accepts and you intend it. Changing an existing exact `.*` fallback requires removing it in one apply and adding its new declaration in another; Typesense rejects a one-request replacement.

This provider release targets Typesense 30.2. Upgrade a Typesense 29.1 server before relying on the collection alteration behavior described here.

## Supported versions and state

The required acceptance matrix is Terraform 1.15/1.16 with Typesense 29.1/30.2. Typesense 29.1 is retained for compatibility testing, but 30.2 is the supported server target; passing tests do not establish 29.1 support. After updating the configuration, refresh existing resources without manually editing state or re-importing them.

Existing state gains resource identity metadata on refresh; resource addresses remain unchanged. Optional `timeouts` settings control operation deadlines. Changing them does not rotate keys or replace collections. See each resource's reference page for timeout defaults.
