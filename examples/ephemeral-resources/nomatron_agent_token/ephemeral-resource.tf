ephemeral "nomatron_agent_token" "example" {
  agent_id        = nomatron_agent.example.id
  name            = "terraform"
  revoke_existing = true
}
