action "nomatron_import_nomad_job" "example" {
  config {
    org_name = "platform"
    app_slug = "payments"
    job_slug = "web"
    job_id   = "payments-web"
  }
}
