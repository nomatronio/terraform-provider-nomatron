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

resource "nomatron_job" "directory" {
  org_name         = "platform"
  app_slug         = "payments"
  name             = "Worker"
  slug             = "worker"
  jobspec_type     = "hcl"
  source_mode      = "directory"
  source_directory = "jobs/worker"
  job_file_path    = "jobs/worker/job.nomad.hcl"
  job_var_file_paths = [
    "jobs/worker/base.hcl",
    "jobs/worker/generated.json",
  ]
  default_namespace = "payments"
  priority          = 2
}
