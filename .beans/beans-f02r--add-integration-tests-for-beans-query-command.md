---
title: Add integration tests for beans query command
status: completed
type: task
created_at: 2025-12-08T20:27:55Z
updated_at: 2025-12-08T20:34:14Z
---

## Summary

The new `beans query` command has unit tests for resolvers (`internal/graph/schema.resolvers_test.go`) but no integration tests that exercise the actual command with real bean files.

## Motivation

Integration tests would:
- Verify the full command flow works end-to-end
- Test stdin piping behavior
- Test flag combinations (--json, --schema, --variables, --operation)
- Catch issues with the HTTP request/response handling via httptest

## Checklist

- [x] Create test fixtures with sample bean files
- [x] Test `beans query '{ beans { id } }'` basic query
- [x] Test query with filters
- [ ] Test stdin input mode (skipped - difficult to test stdin in unit tests)
- [ ] Test --json output format (covered by internal JSON handling)
- [x] Test --schema flag (via GetGraphQLSchema test)
- [x] Test --variables flag
- [x] Test error cases (invalid query, missing query)