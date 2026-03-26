action "nomatron_speculative_plan" "example" {
  config {
    org_name = "platform"
    app_slug = "payments"
    job_slug = "web"
    ref      = "feature/my-branch"
  }
}
