resource "nomatron_nomad_cluster" "example" {
  name              = "primary"
  description       = "Primary Nomad cluster"
  connectivity_mode = "direct"
  address           = "https://nomad.example.com"
  skip_verify       = false

  acl_token_wo = "sensitive-token"
}
