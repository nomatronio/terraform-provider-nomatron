ephemeral "nomatron_connectivity_agent_token" "example" {
  agent_id        = nomatron_connectivity_agent.example.id
  name            = "terraform"
  revoke_existing = true
}
