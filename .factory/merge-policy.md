# Merge policy

This is the intended policy. GitHub does not enforce it yet. The risk-tier check, when added, only reports a tier and sets a label; it does not turn on auto-merge or change branch protection.

## Reviews

- **Codex Code Review** runs automatically when a pull request opens. Read its result on the current commit.
- **Codex Security Review** is on for pull requests. Its result must be clear before any merge, at every tier. For `risk:medium` and `risk:high`, the opener also comments exactly `@codex security review` once the pull request is ready for review, and again after any substantial new push. An empty result is not a clean pass: read the report before treating the review as finished.
- **Security reviewer:** medium and high pull requests also get a human or security-bot review of the final commit before merge.
- **QA** tests release candidates, not individual merges. It is not a per-pull-request merge condition.
- Do not enable repository-wide security scans or continuous monitoring that bill separately.

## Low

May be merged after the test suite, Codex Code Review, Codex Security Review and the free scanners in CI are clear. Auto-merge is not enabled yet; until it is, the maintainer merges.

## Medium

Needs the same checks, the security reviewer, and a manual `@codex security review`. Merge after a 24-hour hold during which the maintainer can block it.

## High

Needs the same checks as medium. The maintainer merges. No auto-merge.

## Breaking changes

A pull request marked `Breaking: yes` (see `.factory/state-compatibility.md`) is never low risk and is never auto-merged. It needs the maintainer's explicit decision on the ticket. At release, **a breaking change bumps the MAJOR semver version**: do not tag a minor or patch release that contains one.

## Always

- Squash merge only, linear history.
- No pull request description or comment contains private links, internal ticket IDs, credentials or other private details. The repository is public.
- Tagging a release is a maintainer action.
