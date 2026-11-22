# Terraform state compatibility

Terraform state belongs to the people who use this provider. A change that makes an existing state unreadable, forces replacement, or changes how an existing configuration behaves is a **breaking change**.

## What counts as breaking

- Removing or renaming a resource, data source, action or attribute.
- Changing an attribute's type, or making an optional attribute required.
- Adding `RequiresReplace` or ForceNew to an existing attribute, or changing a default so existing configurations plan a change.
- Changing the format of a resource ID or the import format (for example `buildGroupMemberID` style helpers).
- Changing what a read matches on, so existing resources are reported as missing or recreated.
- Raising the minimum Terraform or server version the provider needs.
- Removing or changing provider configuration arguments or environment variables.

Not breaking: adding an optional attribute, adding a new resource or data source, fixing a read that was wrong without changing IDs or state shape, documentation, tests.

## Rules

1. A ticket that may be breaking says so in a `Breaking: yes` line, and states the user-visible effect and a migration path (state upgrade, `moved` block, or manual steps).
2. A breaking change is never labelled low risk, and is never started by an automated agent without an owner decision on the ticket.
3. If a state shape must change, prefer a schema version bump with a `StateUpgrader` that keeps existing state valid. Test the upgrade from a real old state.
4. Never edit state files, run `terraform state rm`, `terraform destroy`, or run `apply` against a real or shared environment from an agent. Use `httptest` servers and the Terraform plugin testing harness.
5. Deprecate first where you can: keep the old attribute working, mark it deprecated in the schema, and remove it only in a later major release.

## Versioning (hard rule)

**A breaking change bumps the MAJOR semver version at release.** A release that contains any breaking change is not released as a minor or patch version. Release notes for it list each breaking change and the migration path, and put them first. Non-breaking additions bump minor; fixes bump patch.

The pull request description repeats `Breaking: yes` or `Breaking: no`. Reviewers and the release step check that a `Breaking: yes` change is held for a major release.
