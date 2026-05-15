data "nomatron_job_approval_policy" "example" {
  org_name = "platform"
  app_slug = "payments"
  job_slug = "web"
}
