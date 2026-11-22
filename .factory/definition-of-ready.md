# Definition of ready

A ticket moves to Ready for Codex only when every prerequisite it names is proved, not assumed. An automated run that stops on a missing prerequisite wastes time and usage.

## Prerequisites section

Every ticket has a `## Prerequisites` section. Write `none` only after checking each category: files on the default branch, merged pull requests and tickets (including in the server project this provider depends on), secrets (named only, never their values), environment and configuration, external setup, owner decisions, and data or content checks. Each prerequisite has:

- **What:** the thing that must be true, in one sentence.
- **Where:** the exact repository and files, or the setting and its location.
- **Delivered by:** the pull request, commit or ticket that provides it. For external setup, the person or step.
- **Verify:** a command or check anyone can run, with the expected result.
- **Owner:** who is responsible.
- **Status:** met, not met, or blocked, with the date checked.

A prerequisite delivered by a pull request counts as met only after it is merged and, where the provider needs a released server, after that release is live.

## Bugs

Every bug ticket has a `## Steps to reproduce` section with:

1. The build or commit it was seen on, and the server version.
2. The setup (provider configuration without secrets, Terraform version, config, data).
3. Numbered, exact steps (`terraform` commands, API requests).
4. Expected result and actual result, with exact error text.
5. A failing test where possible (name and path), or why one is not possible.

A bug missing the section, or any of items 1 to 4, is not ready.

## Breaking changes

The ticket states `Breaking: yes` or `Breaking: no` (see `.factory/state-compatibility.md`). A `Breaking: yes` ticket needs an owner decision recorded on it before it is ready, and its release is a major version bump.

## Readiness check

Before a ticket moves to Ready for Codex:

1. Read the Prerequisites section and the stop checks.
2. Fetch the default branch and verify every prerequisite with its Verify step.
3. For each merged prerequisite pull request, compare its changed files with the ticket's list. Missing or extra files fail the check.
4. Run every stop check.
5. For a bug, read `## Steps to reproduce` against the list above.
6. Post a `Readiness check` comment on the ticket listing each prerequisite, stop check and (for a bug) the steps section as `pass` or `fail`, with the evidence and date.
7. Any `fail`, or any prerequisite still waiting on a person, keeps the ticket out of Ready.

Repeat the check if the default branch changes in a way that touches a prerequisite, or if the ticket has waited more than a day.

Related: `.factory/ticket.md` is the ticket shape. `.agents/skills/scope-ticket/SKILL.md` writes the scope. Risk tiers are in `.factory/risk-paths.txt`.
