resource "nomatron_github_app_integration" "example" {
  name            = "primary"
  app_id          = "12345"
  app_slug        = "nomatron-app"
  client_id       = "Iv1.1234567890abcdef"
  private_key_pem_wo = <<-EOT
  -----BEGIN RSA PRIVATE KEY-----
  ...
  -----END RSA PRIVATE KEY-----
  EOT
  webhook_secret_wo = "super-secret"
}
