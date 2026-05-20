data "nomatron_job" "example" {
  org_name = "platform"
  app_slug = "payments"
  slug     = "web"
}

output "job_repository" {
  value = data.nomatron_job.example.effective_repo_url
}
