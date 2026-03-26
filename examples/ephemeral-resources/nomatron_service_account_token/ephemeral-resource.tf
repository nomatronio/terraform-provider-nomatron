ephemeral "nomatron_service_account_token" "example" {
  service_account_id = nomatron_service_account.example.id
  name               = "terraform"
  expires_at         = "2026-12-31T23:59:59Z"
}
