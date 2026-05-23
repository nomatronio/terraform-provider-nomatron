data "nomatron_job" "example" {
  org_name = "platform"
  app_slug = "payments"
  slug     = "web"
}

output "job_repository" {
  value = data.nomatron_job.example.effective_repo_url
}

output "job_source_mode" {
  value = data.nomatron_job.example.source_mode
}

output "job_var_files" {
  value = data.nomatron_job.example.job_var_file_paths
}
