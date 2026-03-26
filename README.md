# Terraform Provider for Nomatron

Terraform provider for managing Nomatron resources, lookups, ephemeral secrets, and operational actions.

## Requirements

- Terraform `>= 1.14` to use actions
- Terraform `>= 1.11` to use write-only resource attributes such as `value_wo`

## Provider Configuration

```hcl
terraform {
  required_providers {
    nomatron = {
      source = "nomatronio/nomatron"
    }
  }
}

provider "nomatron" {
  address = "https://nomatron.example.com"
  token   = var.nomatron_token
}
```

Optional TLS arguments:

- `tls_cert`
- `tls_key`
- `tls_ca_cert`

## Quick Start

```hcl
provider "nomatron" {
  address = "https://nomatron.example.com"
  token   = var.nomatron_token
}

resource "nomatron_organization" "platform" {
  name = "platform"
}

resource "nomatron_agent" "runner" {
  name        = "platform-runner"
  description = "Terraform managed runner"
}
```

## Supported Resources

- `nomatron_agent`
- `nomatron_application`
- `nomatron_environment`
- `nomatron_github_app_integration`
- `nomatron_group`
- `nomatron_group_member`
- `nomatron_job`
- `nomatron_nomad_cluster`
- `nomatron_organization`
- `nomatron_organization_github_app_integration`
- `nomatron_organization_member`
- `nomatron_organization_nomad_cluster`
- `nomatron_role`
- `nomatron_role_assignment`
- `nomatron_service_account`
- `nomatron_user`
- `nomatron_variable`

Examples live under [examples/resources](/Users/robertbarnes/Developer/terraform-provider-nomatron/examples/resources).

## Supported Data Sources

- `nomatron_agent`
- `nomatron_application`
- `nomatron_environment`
- `nomatron_github_app_integration`
- `nomatron_group`
- `nomatron_group_member`
- `nomatron_job`
- `nomatron_nomad_cluster`
- `nomatron_organization`
- `nomatron_organization_github_app_integration`
- `nomatron_organization_member`
- `nomatron_organization_nomad_cluster`
- `nomatron_role`
- `nomatron_service_account`
- `nomatron_users`

Examples live under [examples/data-sources](/Users/robertbarnes/Developer/terraform-provider-nomatron/examples/data-sources).

## Supported Ephemeral Resources

- `nomatron_agent_token`
- `nomatron_service_account_token`

Examples live under [examples/ephemeral-resources](/Users/robertbarnes/Developer/terraform-provider-nomatron/examples/ephemeral-resources).

## Supported Actions

- `nomatron_import_nomad_job`
- `nomatron_speculative_plan`

Examples live under [examples/actions](/Users/robertbarnes/Developer/terraform-provider-nomatron/examples/actions).

## Variable Resource Notes

`nomatron_variable` supports four Terraform-facing scopes:

- `global`
- `organization`
- `app`
- `job`

For default values:

- Use `value` for non-sensitive variables so the default value is stored in Terraform state.
- Use `value_wo` for sensitive variables so the value is write-only and not stored in plan or state.
- `value` and `value_wo` are mutually exclusive.

Example:

```hcl
resource "nomatron_variable" "db_password" {
  scope     = "job"
  org_name  = "platform"
  app_slug  = "payments"
  job_slug  = "web"
  key       = "db_password"
  sensitive = true
  value_wo  = var.db_password
}
```

## Importing Resources

Most managed resources support `terraform import`.

Common patterns:

- UUID-based imports:
  - `nomatron_agent`
  - `nomatron_user`
  - `nomatron_service_account`
  - `nomatron_role`
  - `nomatron_organization`

- Scoped imports:
  - `nomatron_application`: `org_name/app_slug`
  - `nomatron_job`: `org_name/app_slug/job_slug`
  - `nomatron_environment`: `org_name/app_slug/job_slug/environment_slug`
  - `nomatron_organization_nomad_cluster`: `org_name/name`
  - `nomatron_organization_github_app_integration`: `org_name/name`
  - `nomatron_variable`:
    - `global/<variable_id>`
    - `organization/<org_name>/<variable_id>`
    - `app/<org_name>/<app_slug>/<variable_id>`
    - `job/<org_name>/<app_slug>/<job_slug>/<variable_id>`

Examples:

```hcl
terraform import nomatron_agent.example aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
terraform import nomatron_application.example platform/payments
terraform import nomatron_environment.example platform/payments/web/prod
terraform import nomatron_variable.example job/platform/payments/web/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
```

## Notes and Limitations

- `nomatron_role_assignment` currently has create/delete support, but the API/SDK does not yet expose a full read/list role-assignment endpoint. The resource can still be managed, but drift detection is limited and there is not currently a matching data source.
- Terraform actions are imperative operations. They can emit progress and diagnostics, but they do not return structured outputs into configuration the way resources or data sources do.
- Sensitive Terraform attributes marked write-only are not stored in Terraform plan or state, but they require a Terraform version that supports write-only resource attributes.

## Examples

See the full example set in:

- [examples/resources](/Users/robertbarnes/Developer/terraform-provider-nomatron/examples/resources)
- [examples/data-sources](/Users/robertbarnes/Developer/terraform-provider-nomatron/examples/data-sources)
- [examples/ephemeral-resources](/Users/robertbarnes/Developer/terraform-provider-nomatron/examples/ephemeral-resources)
- [examples/actions](/Users/robertbarnes/Developer/terraform-provider-nomatron/examples/actions)

## Development

Run provider tests with:

```bash
GOCACHE=/Users/robertbarnes/Developer/terraform-provider-nomatron/.gocache \
GOMODCACHE=/Users/robertbarnes/Developer/terraform-provider-nomatron/.gomodcache \
go test ./internal/provider/...
```
