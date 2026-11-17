<a name="unreleased"></a>
## [Unreleased]


<a name="v1.12.0"></a>
## [v1.12.0] - 2026-11-17
### Chore
- **ci:** test Terraform 1.0 and the three latest minor versions ([#139](https://github.com/i3dnet/terraform-provider-i3dnet/issues/139))
- **network:** Re-vendor the Network API spec at networkapi-core v0.55.8 ([#146](https://github.com/i3dnet/terraform-provider-i3dnet/issues/146))

### Docs
- changelog for v1.7.1 to v1.11.0 ([#145](https://github.com/i3dnet/terraform-provider-i3dnet/issues/145))

### Features
- **flexmetal-server:** networks on i3dnet_flexmetal_server (FM-5404) ([#144](https://github.com/i3dnet/terraform-provider-i3dnet/issues/144))


<a name="v1.11.0"></a>
## [v1.11.0] - 2026-10-16
### Features
- **network:** i3dnet_network_ip_addresses data source (NISE-4308, PR 5) ([#137](https://github.com/i3dnet/terraform-provider-i3dnet/issues/137))
- **network:** i3dnet_network_prefixes data source (NISE-4210, PR 4b) ([#136](https://github.com/i3dnet/terraform-provider-i3dnet/issues/136))
- **network:** i3dnet_network and i3dnet_networks data sources (NISE-4210, PR 4a) ([#135](https://github.com/i3dnet/terraform-provider-i3dnet/issues/135))
- **network:** generated Network API client and i3dnet_fabrics data source (NISE-4210, PR 1/4) ([#132](https://github.com/i3dnet/terraform-provider-i3dnet/issues/132))


<a name="v1.10.6"></a>
## [v1.10.6] - 2026-09-09
### Bug Fixes
- **flexmetal-server:** retry a refresh whose GET never completed ([#131](https://github.com/i3dnet/terraform-provider-i3dnet/issues/131))


<a name="v1.10.5"></a>
## [v1.10.5] - 2026-09-03
### Bug Fixes
- **flexmetal-server:** populate name, location, instance_type and os on import [FM-4961] ([#128](https://github.com/i3dnet/terraform-provider-i3dnet/issues/128))

### Features
- **flexmetal-server:** ramp the delivery poll interval down to 15s ([#129](https://github.com/i3dnet/terraform-provider-i3dnet/issues/129))


<a name="v1.10.4"></a>
## [v1.10.4] - 2026-09-02
### Bug Fixes
- **flexmetal-server:** adopt servers whose create request timed out [FM-4949] ([#118](https://github.com/i3dnet/terraform-provider-i3dnet/issues/118))

### Chore
- **ci:** move to golangci-lint v2 and bump the action majors ([#120](https://github.com/i3dnet/terraform-provider-i3dnet/issues/120))
- **deps:** clear the security alerts and bump the plugin SDK ([#119](https://github.com/i3dnet/terraform-provider-i3dnet/issues/119))


<a name="v1.10.3"></a>
## [v1.10.3] - 2026-07-16
### Bug Fixes
- **flexvm-vm:** keep omitted description null when API returns "" ([#115](https://github.com/i3dnet/terraform-provider-i3dnet/issues/115))


<a name="v1.10.2"></a>
## [v1.10.2] - 2026-07-15
### Bug Fixes
- **flexvm-node:** bump default create timeout to 45m ([#114](https://github.com/i3dnet/terraform-provider-i3dnet/issues/114))


<a name="v1.10.1"></a>
## [v1.10.1] - 2026-06-29
### Features
- **flexvm-vm:** support tags on VM create ([#110](https://github.com/i3dnet/terraform-provider-i3dnet/issues/110))


<a name="v1.10.0"></a>
## [v1.10.0] - 2026-06-29
### Features
- add `user_data_file` field, to add custom cloud-init configuration ([#104](https://github.com/i3dnet/terraform-provider-i3dnet/issues/104))
- **flexvm-cloud:** add flexvm_cloud resource ([#106](https://github.com/i3dnet/terraform-provider-i3dnet/issues/106))
- **flexvm-node:** add flexvm_node resource ([#109](https://github.com/i3dnet/terraform-provider-i3dnet/issues/109))
- **flexvm-node:** add `flexvm_node` and `flexvm_nodes` data sources ([#108](https://github.com/i3dnet/terraform-provider-i3dnet/issues/108))
- **flexvm_cloud:** add cloud as a data source ([#107](https://github.com/i3dnet/terraform-provider-i3dnet/issues/107))


<a name="v1.9.1"></a>
## [v1.9.1] - 2026-05-19
### Bug Fixes
- **flexvm-vm:** do not remove "failed" state + handle delete responses properly ([#103](https://github.com/i3dnet/terraform-provider-i3dnet/issues/103))


<a name="v1.9.0"></a>
## [v1.9.0] - 2026-04-13
### Features
- **flexvm:** add flex vm resource ([#100](https://github.com/i3dnet/terraform-provider-i3dnet/issues/100))


<a name="v1.8.1"></a>
## [v1.8.1] - 2026-02-11
### Bug Fixes
- updated documentation links ([#91](https://github.com/i3dnet/terraform-provider-i3dnet/issues/91))

### Chore
- goreleaser - update deprecated "archive.format" to "archive.formats". ([#90](https://github.com/i3dnet/terraform-provider-i3dnet/issues/90))


<a name="v1.8.0"></a>
## [v1.8.0] - 2026-02-10
### Chore
- bump dependency versions. ([#84](https://github.com/i3dnet/terraform-provider-i3dnet/issues/84))

### Features
- support for custom-ipxe requests. ([#86](https://github.com/i3dnet/terraform-provider-i3dnet/issues/86))


<a name="v1.7.1"></a>
## [v1.7.1] - 2025-10-31
### Chore
- Updated documentation and fixed broken links. Updated dependencies. ([#82](https://github.com/i3dnet/terraform-provider-i3dnet/issues/82))


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


[Unreleased]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.12.0...HEAD
[v1.12.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.11.0...v1.12.0
[v1.11.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.10.6...v1.11.0
[v1.10.6]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.10.5...v1.10.6
[v1.10.5]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.10.4...v1.10.5
[v1.10.4]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.10.3...v1.10.4
[v1.10.3]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.10.2...v1.10.3
[v1.10.2]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.10.1...v1.10.2
[v1.10.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.10.0...v1.10.1
[v1.10.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.9.1...v1.10.0
[v1.9.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.9.0...v1.9.1
[v1.9.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.8.1...v1.9.0
[v1.8.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.8.0...v1.8.1
[v1.8.0]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.7.1...v1.8.0
[v1.7.1]: https://github.com/i3dnet/terraform-provider-i3dnet/compare/v1.7.0...v1.7.1
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
