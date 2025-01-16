# i3D.net Terraform Provider

The `i3D.net Terraform Provider` allows you to manage [i3D.net](https://www.i3d.net/) resources using Terraform.

- Documentation: https://registry.terraform.io/providers/i3D-net/i3dnet/latest/docs

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up-to-date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

See the [i3D.net Provider documentation](https://registry.terraform.io/providers/i3D-net/i3dnet/latest/docs) to get
started using the i3D.net provider.

## Developing the Provider

See [CONTRIBUTING.md](./CONTRIBUTING.md) for information about contributing to this project.

