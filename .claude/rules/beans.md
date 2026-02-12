## Sorting Consistency

Bean sorting must be consistent across the entire system. All places that return lists of beans should use `bean.SortByStatusPriorityAndType()` from `internal/bean/sort.go`. This includes:

- CLI commands (`beans list`, etc.)
- TUI views
- GraphQL API resolvers (`beans`, `children`, `blockedBy`, `blocking` queries)

The sort order is: status -> priority -> type -> title (case-insensitive).
