---
title: Add integration tests for beans query command
status: backlog
type: task
created_at: 2025-12-08T20:27:55Z
updated_at: 2025-12-08T20:27:55Z
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

- [ ] Create test fixtures with sample bean files
- [ ] Test `beans query '{ beans { id } }'` basic query
- [ ] Test query with filters
- [ ] Test stdin input mode
- [ ] Test --json output format
- [ ] Test --schema flag
- [ ] Test --variables flag
- [ ] Test error cases (invalid query, missing query)