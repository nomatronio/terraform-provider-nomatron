---
name: security-review
description: Review a medium or high risk provider change. Require one proved fact that makes it safe, or block. Use for provider configuration, authentication, workflows, release and signing, dependency pins, and anything touching credentials or state compatibility.
---

# Security review

Use this for medium and high risk, and for any diff that touches credentials, tokens, workflow permissions, release or signing, or dependency pins. Read `.factory/merge-policy.md` and the diff. Do not approve from the title.

For a pull request labelled `risk:medium` or `risk:high`, comment exactly `@codex security review` once it is ready for review, and again after any substantial new push. A clear Codex Security Review is required before merge at every tier. An empty result is not a clean pass.

## Blast radius

This step is required.

1. Name the asset a mistake would expose or widen: an API token handled by the provider, Terraform state contents, a CI secret, a signing key, or users' existing state.
2. State the one fact that makes the change safe in one sentence: "This is safe because ...".
3. Prove it from the diff or a test you ran. Name the file.
4. If you cannot prove it, block and name the missing proof.

## Also look for

- Tokens, passwords or secrets written to logs, error messages, diagnostics, state or test fixtures. Sensitive schema attributes must be marked `Sensitive`.
- TLS verification turned off, or an endpoint that is not validated.
- A workflow whose token permissions grew, a `pull_request_target` trigger, a secret exposed to code from a fork, an unpinned third-party action, or a changed release or signing step.
- A dependency bump in `go.mod` or `go.sum` that nobody asked for.
- A change that is breaking for existing state but is not flagged `Breaking: yes`. Breaking changes need an owner decision and a major version bump.
- Private details in a public repository: private links, internal ticket IDs, customer names, credentials.

Finish with `safe` and the proved fact, or `block` and the missing proof.
