resource "nomatron_service_account" "example" {
  name        = "terraform"
  description = "Automation service account"
  is_active   = true
}
