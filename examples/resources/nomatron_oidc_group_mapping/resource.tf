resource "nomatron_oidc_group_mapping" "engineering_root" {
  provider_slug  = nomatron_oidc_provider.google.slug
  external_group = "Engineering"
  role           = "root"
  domain         = "global"
  description    = "Engineering group from Google Workspace"
  enabled        = true
}

