resource "nomatron_application" "example" {
  org_name     = "platform"
  name         = "Payments"
  slug         = "payments"
  cluster_id   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  repo_url     = "https://github.com/nomatron/payments"
  git_provider = "github"
  trigger_mode = "branch_commit"
  ref          = "main"
  auto_plan    = true
  auto_apply   = false
}

resource "nomatron_application" "tagged_release" {
  org_name         = "platform"
  name             = "Payments Tagged Release"
  slug             = "payments-tagged-release"
  cluster_id       = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
  repo_url         = "https://github.com/nomatron/payments"
  git_provider     = "github"
  trigger_mode     = "tag"
  tag_pattern_type = "prefix"
  tag_pattern      = "v-"
  auto_plan        = true
  auto_apply       = false
}
