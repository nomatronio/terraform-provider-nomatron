resource "nomatron_group" "example" {
  org_name       = nomatron_organization.example.name
  name           = "admins"
  description    = "Platform administrators"
  owner_username = "rbarnes"

  metadata = {
    team = "platform"
  }
}
