resource "nomatron_organization_nomad_cluster" "example" {
  org_name          = "platform"
  name              = "primary"
  description       = "Primary Nomad cluster for the platform org"
  connectivity_mode = "direct"
  address           = "https://nomad.example.com"
  skip_verify       = false

  acl_token = "sensitive-token"
}
