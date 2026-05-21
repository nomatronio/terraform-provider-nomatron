action "nomatron_destroy_app_jobs" "payments" {
  config {
    org_name = "acme"
    app_slug = "payments"

    # Queues a destroy plan for every live Nomad job in this app.
    # This does not delete Nomatron application or job resources.
    apply = false
  }
}
