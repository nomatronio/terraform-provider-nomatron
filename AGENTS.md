# Terraform provider for Nomatron

This repository is the Terraform provider for Nomatron, a deployment control plane for HashiCorp Nomad. It is written in Go with the Terraform Plugin Framework. The repository is **public**: everything you write here, in a pull request, a commit message, a test fixture or a comment, is public.

## Public repository rule (hard)

Never put any of these in a file, commit, pull request, issue or comment: private repository links or paths, internal ticket or tracker links or IDs, internal email addresses, customer names, pricing, credentials or secret values, local machine paths, internal hostnames, or the licence server host. Say "the maintainers' tracker" if you must refer to one. Test fixtures use `example.com` style data only.

## Layout

- `main.go` starts the provider server.
- `internal/provider/` holds everything: `provider.go` and `client.go` (configuration and authentication), `*_resource.go`, `*_data_source.go`, `*_ephemeral_resource.go` and `*_action.go`, each with a `*_test.go` next to it.
- `docs/` is generated from the schema by `make docs` (tfplugindocs), using `templates/` and `examples/`. Do not hand-edit generated docs; change the schema description or the template, then regenerate.
- The provider talks to the Nomatron API through the generated Go SDK pinned in `go.mod`. Do not bump or regenerate the SDK unless the ticket says so.
- `goreleaser.yml` and `.github/workflows/` build, sign and release. Off limits unless the ticket names them.

## Build and test

Go version is in `go.mod`.

- Format: `gofmt -l .` must print nothing.
- Vet: `go vet ./...`
- Tests: `go test ./... -count=1`. Unit tests use `httptest` and need no secrets. Acceptance tests need a running Nomatron server; if you did not run them, say so in the pull request.
- Build: `go build ./...`
- Docs: `make docs`. Commit only the doc files your change affects.

## Conventions

- Name the surfaces before coding: resource schema, state and ID format, import format, data source, docs, examples, tests, release notes.
- Write a failing test first when a meaningful one exists.
- Stay inside the files the ticket lists.
- Summarise rather than paste large output.
- Skills in `.agents/skills/` load on demand: `scope-ticket`, `implement-ticket`, `security-review`, `release-notes`.
- Tickets: `.factory/ticket.md` is the shape. Every ticket passes `.factory/definition-of-ready.md` before implementation starts.
- When facts conflict, trust code and tests over docs or tickets.

## State compatibility (hard rule)

Terraform state is the user's data. Changing an attribute, an ID or an import format can break existing users' state. The full rule is `.factory/state-compatibility.md`. A breaking change must be flagged on the ticket and the pull request, and it ships only in a **major** version bump.

## Risk tiers

`.factory/risk-paths.txt` is the path list; `.factory/risk_tier.py` classifies a change. Highest match wins. Unmatched files are medium.

- **Low.** Examples and README.
- **Medium.** Resource, data source, action and ephemeral resource behaviour; tests; generated docs; agent skills and editor rules, except the security-review skill.
- **High.** Provider configuration and authentication, workflows, release and signing files, dependency pins (`go.mod`, `go.sum`, `go.work`, `vendor/`), `.config/`, `.factory/`, the security-review skill, anything that changes state compatibility.

Merge policy: `.factory/merge-policy.md`.

## Handoff

Report surfaces finished, commands actually run, the risk tier, whether the change is breaking, and anything you could not verify.
