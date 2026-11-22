---
name: scope-ticket
description: Turn a ticket into a small, explicit scope before any code change in the Terraform provider. Use at the start of implementation work.
---

# Scope a ticket

Do this before editing code. Do not implement in this step.

1. Restate the outcome in one sentence a Terraform user would recognise.
2. List the surfaces the change touches: resource schema, state / ID / import format, data source, action or ephemeral resource, provider configuration, generated docs, examples, tests, release notes. Drop the ones it does not touch.
3. Read `.factory/risk-paths.txt` and name the tier from the files you expect to change. Highest match wins.
4. Decide `Breaking: yes` or `Breaking: no` using `.factory/state-compatibility.md`. A breaking change needs an owner decision and ships in a major version bump. Do not scope it as low risk.
5. Name the files to edit and the files to leave alone.
6. Write the test that would prove the outcome (usually an `httptest` server), or say why none is meaningful.
7. Write a release-note line only when a user would notice; otherwise `no release note`.
8. Write the `## Prerequisites` section, stop checks and `## Skills to use` as `.factory/definition-of-ready.md` describes. Verify each prerequisite against the default branch; do not mark one met from a title.
9. For a bug, write `## Steps to reproduce` as `.factory/ticket.md` shows.

Use `.factory/ticket.md` as the shape. Keep it short. Split a broad epic into one change you can finish.

The repository is public: no private links, internal ticket IDs, credentials or customer names in the ticket text that will become a pull request.

Summarise rather than paste large output.
