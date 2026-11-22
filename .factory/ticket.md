# Ticket

## Outcome

One sentence a user of the provider would recognise.

## Prerequisites

Each entry has what, where, delivered by, verify, owner and status. See `.factory/definition-of-ready.md`. Write `none` only after checking each category.

- What:
- Where:
- Delivered by:
- Verify:
- Owner:
- Status:

## Surfaces

- [ ] resource schema
- [ ] state / ID / import format
- [ ] data source
- [ ] action or ephemeral resource
- [ ] provider configuration / authentication
- [ ] docs (generated)
- [ ] examples
- [ ] tests
- [ ] release notes

## Risk tier

`low`, `medium` or `high`. List the paths that set the tier. Highest match in `.factory/risk-paths.txt` wins.

## Breaking

`Breaking: yes` or `Breaking: no`. See `.factory/state-compatibility.md`.

If yes: the user-visible effect, the migration path, and the owner decision. **A breaking change bumps the MAJOR semver version at release.** It is never low risk, and it is not started by an automated agent without an owner decision recorded here.

## Non-goals

What this change will not do.

## State safety

Does any schema attribute, ID format or import format change? If yes, the ticket is `Breaking: yes` unless a state upgrader keeps existing state valid. Agents never run `apply`, `destroy` or state commands against a real environment.

## Stop checks

Every condition under which the implementer must stop. Each is run before the ticket moves.

## Test

The failing test to write first, or why no meaningful test can be written.

## Off limits

Files and areas the change must not touch.

## Skills to use

Name the skills the implementer should use (for example `scope-ticket`, `implement-ticket`, `security-review`, `release-notes`).

## Release note

One sentence for the release notes, or `no release note`. A breaking change names the migration path and is held for a major release.

## Bugs only

Include `## Steps to reproduce` for a bug (see the definition of ready). Omit it otherwise.

## Steps to reproduce

1. **Seen on:** provider version or commit SHA, and server version.
2. **Setup:** Terraform version, configuration without secrets, data needed.
3. **Steps:**
   1. Exact command or request.
4. **Expected:**
   **Actual:** exact error text or output.
5. **Failing test:** name and path, or why one is not possible.
