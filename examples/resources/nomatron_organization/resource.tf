resource "nomatron_organization" "example" {
  name           = "platform"
  description    = "Platform engineering"
  owner_username = "jdoe"

  metadata = {
    team  = "platform"
    owner = "terraform"
  }
}
