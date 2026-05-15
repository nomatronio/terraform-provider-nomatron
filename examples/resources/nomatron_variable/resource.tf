resource "nomatron_variable" "example" {
  scope     = "job"
  org_name  = "platform"
  app_slug  = "payments"
  job_slug  = "web"
  key       = "db_password"
  sensitive = true
  value_wo  = "super-secret-password"

  environment_values = [
    {
      environment_slug = "dev"
      value_wo         = "dev-secret-password"
    },
    {
      environment_slug = "prod"
      value_wo         = "prod-secret-password"
    }
  ]
}
