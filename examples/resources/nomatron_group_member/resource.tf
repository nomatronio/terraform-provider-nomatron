resource "nomatron_group_member" "example" {
  org_name   = nomatron_organization.example.name
  group_name = nomatron_group.example.name
  username   = "rbarnes"
}
