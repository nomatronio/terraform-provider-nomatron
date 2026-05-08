resource "nomatron_oidc_provider" "google" {
  slug             = "google"
  display_name     = "Google Workspace"
  issuer_url       = "https://accounts.google.com"
  client_id        = "google-client-id.apps.googleusercontent.com"
  client_secret_wo = var.google_oidc_client_secret

  scopes         = ["openid", "profile", "email"]
  username_claim = "email"
  email_claim    = "email"
  name_claim     = "name"
  groups_claim   = "groups"

  allowed_email_domains = ["example.com"]
  enabled               = true
  is_default            = true
  auto_provision        = true
  sync_profile          = true
  sync_groups           = true
}

