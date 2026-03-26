resource "nomatron_variable" "example" {
  scope     = "job"
  org_name  = "platform"
  app_slug  = "payments"
  job_slug  = "web"
  key       = "db_password"
  sensitive = true
  value_wo  = "super-secret-password"
}
