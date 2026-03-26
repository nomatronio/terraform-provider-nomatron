resource "nomatron_organization_member" "example" {
  org_name = nomatron_organization.example.name
  username = "rbarnes"
}
