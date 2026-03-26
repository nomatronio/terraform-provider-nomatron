resource "nomatron_role" "example" {
  name        = "viewer"
  description = "Read-only role"
  permissions = [
    "global.roles.read",
    "global.users.read",
  ]
}
