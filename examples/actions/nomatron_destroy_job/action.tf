action "nomatron_destroy_job" "web" {
  config {
    org_name = "acme"
    app_slug = "payments"
    job_slug = "web"

    # Set apply=true only when the destroy plan is ready for apply.
    # Approval-required plans must be approved in Nomatron before apply can be queued.
    apply = false
  }
}
