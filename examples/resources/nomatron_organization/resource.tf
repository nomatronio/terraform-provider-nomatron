resource "nomatron_organization" "example" {
  name          = "platform"
  description   = "Platform engineering"
  owner_username = "rbarnes"

  metadata = {
    team  = "platform"
    owner = "terraform"
  }
}
