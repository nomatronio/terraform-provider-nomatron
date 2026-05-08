data "nomatron_oidc_group_mapping" "engineering_root" {
  provider_slug  = "google"
  external_group = "Engineering"
  role           = "root"
  domain         = "global"
}
