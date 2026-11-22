## Summary

What changed, in a few sentences.

## Risk tier

`low`, `medium` or `high`, and the paths that set it.

Medium and high need the security-review blast-radius step: state the one fact that makes the change safe and where it is proved, or say `block` and what is missing. For `risk:medium` and `risk:high`, comment exactly `@codex security review` once the pull request is ready for review, and again after any substantial new push.

## Breaking

`Breaking: yes` or `Breaking: no`. A breaking change must ship in a MAJOR version bump. See `.factory/state-compatibility.md`.

## State safety

Does any schema attribute, ID format or import format change? Say no, or describe the state upgrade.

## Tests

Commands run and the result in one line each. Say whether acceptance tests ran.

## Release note

A sentence, or `no release note`.
