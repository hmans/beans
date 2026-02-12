---
# beans-qq9i
title: CLI subscription support for beans query
status: draft
type: feature
created_at: 2025-12-20T09:54:53Z
updated_at: 2025-12-20T09:54:53Z
---

Add subscription support to the `beans query` command so users can subscribe to bean changes from the CLI.

## Current Behavior
`beans query "subscription { beanChanged { type beanId } }"` returns error:
"Schema does not support operation type 'subscription'"

## Proposed Solution
1. Parse query to detect if it's a subscription operation
2. If subscription:
   - Start file watcher (`core.StartWatching()`)
   - Execute subscription resolver to get event channel
   - Stream events to stdout as JSON lines until Ctrl+C
3. Handle graceful cleanup on interrupt

## Expected UX
```bash
$ beans query "subscription { beanChanged { type beanId } }"
# Watching for bean changes... (Ctrl+C to stop)
{"type":"CREATED","beanId":"abc123","bean":{...}}
{"type":"UPDATED","beanId":"abc123","bean":{...}}
^C
```

## Use Cases
- Piping to other tools (`beans query "subscription {...}" | jq ...`)
- Debugging/monitoring bean changes
- Integration with external systems

## Files
- cmd/graphql.go