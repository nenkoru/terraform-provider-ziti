---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ziti Provider"
subcategory: ""
description: |-
  
---

# ziti Provider



## Example Usage

```terraform
provider "ziti" {
  username      = "testadmin"
  password      = "testadmin"
  mgmt_endpoint = "https://localhost:1280/edge/management/v1"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `capool` (String) A base64 encoded CA Pool of the Edge Management API.
- `mgmt_endpoint` (String) An endpoint pointing to Ziti Edge Management API URL
- `password` (String, Sensitive) A password of an identity that is able to perform admin actions
- `username` (String) A username of an identity that is able to perform admin actions
