resource "nomatron_application" "example" {
  org_name     = "platform"
  name         = "Payments"
  slug         = "payments"
  cluster_id   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  repo_url     = "https://github.com/nomatron/payments"
  git_provider = "github"
  ref          = "main"
  auto_plan    = true
  auto_apply   = false
}
