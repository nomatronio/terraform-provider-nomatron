resource "nomatron_job" "example" {
  org_name          = "platform"
  app_slug          = "payments"
  name              = "Web"
  slug              = "web"
  jobspec_path      = "jobs/web.nomad.hcl"
  jobspec_type      = "hcl"
  default_namespace = "payments"
  is_primary        = true
}
