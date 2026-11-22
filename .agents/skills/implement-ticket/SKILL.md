---
name: implement-ticket
description: Implement a provider ticket that already has a scope. Write a failing test first, or skip it when a meaningful one cannot be written.
---

# Implement a ticket

Start from a finished scope. Stay inside it and inside the files the ticket lists.

## Failing test first

1. Name the behaviour that is missing or wrong on the current code.
2. Add the smallest test that fails for that reason (an `httptest` server for API behaviour). Run it and read the failure.
3. Change the code until it passes. Run `go test ./... -count=1`.
4. Run `gofmt -l .`, `go vet ./...` and `go build ./...`.
5. If you changed a schema description or template, run `make docs` and commit only the doc files your change affects.

## When to skip the test

Skip it for wording-only edits, generated files you must not hand-edit, or a check that would only assert a mock you control. Say why in the pull request. Do not add a test that already passes or that asserts a constant you just introduced.

## State safety

Follow `.factory/state-compatibility.md`. Do not change a schema attribute, ID format or import format unless the ticket says so. If the change turns out to be breaking and the ticket says `Breaking: no`, stop and report. Never run `apply`, `destroy` or state commands against a real environment.

## Pull request

One branch, one pull request, do not merge it. Use `.factory/pull_request.md`. Open it as ready for review only when tests pass. No private details, internal ticket links or tool footers in the title, body or commits. State whether acceptance tests ran, the risk tier, and `Breaking: yes/no`.

Summarise rather than paste large output.
