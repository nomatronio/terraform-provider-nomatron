resource "nomatron_environment" "example" {
  org_name   = "platform"
  app_slug   = "payments"
  name       = "Production"
  slug       = "prod"
  cluster_id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  namespace  = "payments-prod"
  priority   = 100
}
