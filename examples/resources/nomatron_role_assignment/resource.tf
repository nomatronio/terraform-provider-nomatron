// Note: the current SDK does not expose a role-assignment read/list endpoint,
// so Terraform cannot fully verify remote drift for this resource during refresh.
resource "nomatron_role_assignment" "example" {
  subject = "user:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  role    = "viewer"
  domain  = "global"
}
