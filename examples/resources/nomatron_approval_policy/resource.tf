resource "nomatron_approval_policy" "example" {
  org_name = "platform"
  app_slug = "payments"

  default_rule = {
    required_approvals = 1
    users              = ["alice"]
    groups             = []
  }

  environment_rules = [
    {
      environment_id     = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
      required_approvals = 2
      users              = ["alice", "bob"]
      groups             = ["bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"]
    }
  ]
}
