<a name="unreleased"></a>
## [Unreleased]


<a name="v1.7.0"></a>
## [v1.7.0] - 2025-06-30
### Features
- New flexmetal_server UPDATE behavior when OS parameters are changed in your Terraform configuration. In that case the OS will be re-installed on existing servers, instead of the server resource being destroyed and re-created.

### Bug Fixes
- When applying a plan that had a failed / tainted server in the state file, Terraform wants to delete it before recreating, but it would fail to delete it. Now a failed server will alwyas be destroyed instantly.

### Chore
- **deps:** bump github.com/hashicorp/terraform-plugin-testing ([#64](https://github.com/i3dnet/terraform-provider-i3dnet/issues/64))


<a name="v1.6.3"></a>
## [v1.6.3] - 2025-06-19
### Chore
- renamed all i3d-net references to i3dnet (our new github org)


<a name="v1.6.2"></a>
## [v1.6.2] - 2025-06-13

<a name="v1.6.1"></a>
## [v1.6.1] - 2025-05-16

<a name="1.6.1"></a>
## [1.6.1] - 2025-05-16
### Features
- added changelog configuration and template


<a name="v1.6.0"></a>
## [v1.6.0] - 2025-04-23
### Features
- increase checking for status to 15s


<a name="v1.5.1"></a>
## [v1.5.1] - 2025-04-22
### Bug Fixes
- use correct ErroResponse


<a name="v1.5.0"></a>
## [v1.5.0] - 2025-04-16
### Features
- increase apli client timeout from 20s to 90s


<a name="v1.4.0"></a>
## [v1.4.0] - 2025-04-10
### Features
- running locally in CONTRIBUTING.md
- Added timeouts for create server resource


<a name="v1.3.6"></a>
## [v1.3.6] - 2025-03-18
### Bug Fixes
- go mod tidy
- fix for  returned invalid result object after apply on ssh key rename

### Chore
- go mod fix


<a name="v1.3.5"></a>
## [v1.3.5] - 2025-03-12
### Bug Fixes
- user correct status message for server


<a name="v1.3.4"></a>
## [v1.3.4] - 2025-03-11
### Bug Fixes
- check for ip addresses before saving to state
- return error if server has no ip addresses attached
- if cannot parse error response return generic message with status code

### Features
- add server id to logs
- add extra details to unexpected error


<a name="v1.3.3"></a>
## [v1.3.3] - 2025-03-10
### Features
- dont expose waiting statuses


<a name="v1.3.2"></a>
## [v1.3.2] - 2025-03-10
### Bug Fixes
- check for ErrorResponse before reading .Server

### Chore
- improve method description

### Code Refactoring
- after wait, gets server details to save them to state


<a name="v1.3.1"></a>
## [v1.3.1] - 2025-03-05
### Bug Fixes
- improve errors display

### Code Refactoring
- improve message


<a name="v1.3.0"></a>
## [v1.3.0] - 2025-02-25
### Features
- datasource for locations
- update docs
- add contract_id, overflow; fix waiting for server to be released


<a name="v1.2.3"></a>
## [v1.2.3] - 2025-02-11
### Bug Fixes
- initialize IpAddresses to known value when updating state


<a name="v1.2.2"></a>
## [v1.2.2] - 2025-02-05

<a name="v1.2.1"></a>
## [v1.2.1] - 2025-02-04

<a name="v1.2.0"></a>
## [v1.2.0] - 2025-01-30

<a name="v1.1.0"></a>
## [v1.1.0] - 2025-01-27

<a name="v1.0.5"></a>
## [v1.0.5] - 2025-01-24

<a name="v1.0.4"></a>
## [v1.0.4] - 2025-01-17

<a name="v1.0.3"></a>
## [v1.0.3] - 2025-01-16

<a name="v1.0.2"></a>
## [v1.0.2] - 2025-01-16

<a name="v1.0.1"></a>
## [v1.0.1] - 2025-01-15

<a name="v1.0.0"></a>
## [v1.0.0] - 2025-01-15

<a name="v0.1.0"></a>
## [v0.1.0] - 2024-11-12
### Features
- add binary to gitignore
- add ci cd


<a name="v0.0.0-alpha5"></a>
## [v0.0.0-alpha5] - 2024-11-08
### Features
- not needed the version.


<a name="v0.0.0-alpha4"></a>
## [v0.0.0-alpha4] - 2024-11-08

<a name="v0.0.0-alpha3"></a>
## [v0.0.0-alpha3] - 2024-11-08

<a name="v0.0.0-alpha2"></a>
## [v0.0.0-alpha2] - 2024-11-08
### Bug Fixes
- git repo


<a name="v0.0.0-alpha1"></a>
## [v0.0.0-alpha1] - 2024-11-08
### Bug Fixes
- goreleaser


<a name="v0.0.0-alpha"></a>
## [v0.0.0-alpha] - 2024-11-08
### Bug Fixes
- install jq & yq
- ci

### Features
- add gitlab job
- add ci cd


<a name="v0.0.0"></a>
## v0.0.0 - 2024-11-08
### Features
- Add tags management
- Add initial code for flexmetal provider


[Unreleased]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.7.0...HEAD
[v1.7.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.6.3...v1.7.0
[v1.6.3]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.6.2...v1.6.3
[v1.6.2]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.6.1...v1.6.2
[v1.6.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/1.6.1...v1.6.1
[1.6.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.6.0...1.6.1
[v1.6.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.5.1...v1.6.0
[v1.5.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.5.0...v1.5.1
[v1.5.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.4.0...v1.5.0
[v1.4.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.3.6...v1.4.0
[v1.3.6]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.3.5...v1.3.6
[v1.3.5]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.3.4...v1.3.5
[v1.3.4]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.3.3...v1.3.4
[v1.3.3]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.3.2...v1.3.3
[v1.3.2]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.3.1...v1.3.2
[v1.3.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.3.0...v1.3.1
[v1.3.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.2.3...v1.3.0
[v1.2.3]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.2.2...v1.2.3
[v1.2.2]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.2.1...v1.2.2
[v1.2.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.2.0...v1.2.1
[v1.2.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.0.5...v1.1.0
[v1.0.5]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.0.4...v1.0.5
[v1.0.4]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.0.3...v1.0.4
[v1.0.3]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.0.2...v1.0.3
[v1.0.2]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.0.1...v1.0.2
[v1.0.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v0.1.0...v1.0.0
[v0.1.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v0.0.0-alpha5...v0.1.0
[v0.0.0-alpha5]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v0.0.0-alpha4...v0.0.0-alpha5
[v0.0.0-alpha4]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v0.0.0-alpha3...v0.0.0-alpha4
[v0.0.0-alpha3]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v0.0.0-alpha2...v0.0.0-alpha3
[v0.0.0-alpha2]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v0.0.0-alpha1...v0.0.0-alpha2
[v0.0.0-alpha1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v0.0.0-alpha...v0.0.0-alpha1
[v0.0.0-alpha]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v0.0.0...v0.0.0-alpha
