---
title: Add filtering to list command
status: open
type: feature
---

Add filtering and search capabilities to the `beans list` command.

## New flags for `beans list`

- `--query` / `-q <text>`: Full-text search across bean titles and bodies (case-insensitive)
- `--status` / `-s <status>`: Filter by status (already may exist, verify)
- `--regex`: Treat the query as a regular expression

## Behavior

- Filters can be combined (AND logic)
- Text search is case-insensitive by default
- Results maintain the same output format as unfiltered `list`
- `--json` output continues to work with filters applied

## Example usage

```
beans list -q "authentication"
beans list -s open
beans list -q "login" -s open
beans list -q "auth.*flow" --regex
```

## Checklist

- [ ] Add `-q` / `--query` flag for text search
- [ ] Verify/add `-s` / `--status` flag for status filtering
- [ ] Add `--regex` flag for regex pattern matching
- [ ] Ensure filters work together (AND logic)
- [ ] Update `--json` output to work with filters
- [ ] Update help text
