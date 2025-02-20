<a href="https://terraform.io">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/hashicorp/terraform-provider-aws/refs/heads/main/.github/terraform_logo_dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/hashicorp/terraform-provider-aws/refs/heads/main/.github/terraform_logo_light.svg">
    <img src=".github/terraform_logo_light.svg" alt="Terraform logo" title="Terraform" align="right" height="50">
  </picture>
</a>
<a href="https://openziti.io">
  <picture>
    <img src="https://raw.githubusercontent.com/openziti/ziti-doc/main/docusaurus/static/img/ziti-logo-dark.svg" alt="Terraform logo" title="Terraform" align="right" height="36">
  </picture>
</a>

# Terraform Provider for OpenZiti
***Control your next generation software defined OpenZiti network using Terraform.***

The OpenZiti provider supports all essential resources to start controlling your network.
## Entities and their status of implementation

| Entity                     | Data Source           | Resource            |
|----------------------------|-----------------------|---------------------|
| config                    | ✅                   | ✅                  |
| edge-router-policy        | ✅                   | ✅                  |
| identity                  | ✅                   | ✅                  |
| service                   | ✅                   | ✅                  |
| posture-check             | ✅                   | ✅                  |
| service-policy            | ✅                   | ✅                  |
| service-edge-router-policy| ✅                   | ✅                  |
| auth-policy               | ❌                   | ❌                  |
| authenticator             | ❌                   | ❌                  |
| ca                        | ❌                   | ❌                  |
| edge-router               | ❌                   | ❌                  |
| ext-jwt-signer            | ❌                   | ❌                  |
| terminator                | ❌                   | ❌                  |
| transit-router            | ❌                   | ❌                  |
| config-type               | 🚧                   | 🚧                  |
| enrollment                | 🚧                   | 🚧                  |

🚧 - Enrollment is a one-time thing, barely suitable in Terraform world. Config-type is just beyond the project scope(for now at least).  
✅ - Entity could be fully controlled via a Terraform provider, and that both `one` and `many` datasources are ready to be used.  
❌ - Not yet implemented.  

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22
- [OpenZiti network](https://openziti.io) >= 1.2.1

## Project Goals
- Have a way to control a software-defined OpenZiti network using Terraform

## Project Constraints
- Backwards compatibility of a public interface between minor versions.


## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Example using the provider

```terraform
terraform {
  required_providers {
    ziti = {
      source = "nenkoru/ziti"
    }
  }
}

provider "ziti" {
  username            = "testadmin"
  password        = "testadmin"
  mgmt_endpoint            = "https://localhost:1280/edge/management/v1"
}

resource "ziti_host_config_v1" "simple_host" {
    name = "simple_host.host.v1"
    address = "localhost"
    port    = 5432
    protocol = "tcp"
}
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
