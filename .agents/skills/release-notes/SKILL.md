---
name: release-notes
description: Write or skip a user-facing release note for the provider, and decide the version bump. Use when a change is visible to Terraform users.
---

# Release notes

The provider is released from version tags; release notes live in the GitHub release and, if the repository has one, a changelog file. Match its headings: Breaking changes, Security, Added, Changed, Fixed.

1. Decide if a Terraform user would notice. Refactors and test-only changes get `no release note`.
2. When they would, write one short paragraph under the right heading.
3. Decide the version bump from `.factory/state-compatibility.md`:
   - **Any breaking change bumps the MAJOR version.** Do not release it as a minor or patch. List each breaking change under "Breaking changes", first, with the migration path (state upgrade, `moved` block or manual steps).
   - New resources, data sources or optional attributes bump minor.
   - Fixes bump patch.
4. State the minimum server version when a change needs a newer server.
5. Use the product's words. Do not name customers, prices, internal tickets or internal hosts. The repository is public.
6. Security notes say what the user must do, not how to exploit the old behaviour.

Tagging a release is a maintainer action; propose the version in the pull request, do not tag it.

Keep the paragraph shorter than the diff.
