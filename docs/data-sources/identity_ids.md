---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ziti_identity_ids Data Source - terraform-provider-ziti"
subcategory: ""
description: |-
  A datasource to define a service of Ziti
---

# ziti_identity_ids (Data Source)

A datasource to define a service of Ziti

## Example Usage

```terraform
data "ziti_identity_ids" "test_reference_ziti_identities" {
  filter = "name contains \"test\""
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `filter` (String) ZitiQl filter query

### Read-Only

- `ids` (List of String) An array of allowed addresses that could be forwarded.
