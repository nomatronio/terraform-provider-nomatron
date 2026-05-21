resource "nomatron_job" "example" {
  org_name          = "platform"
  app_slug          = "payments"
  name              = "Web"
  slug              = "web"
  jobspec_path      = "jobs/web.nomad.hcl"
  jobspec_type      = "hcl"
  repo_url          = "https://github.com/acme/payments-web"
  default_namespace = "payments"
  priority          = 1
}
