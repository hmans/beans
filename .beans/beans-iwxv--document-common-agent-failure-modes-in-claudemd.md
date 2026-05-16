---
# beans-iwxv
title: Document common agent failure modes in CLAUDE.md
status: completed
type: task
priority: normal
created_at: 2026-05-16T13:46:35Z
updated_at: 2026-05-16T13:47:12Z
---

Add a 'Common failure modes when invoked by agents' section to CLAUDE.md, distilled from a 307-invocation failure analysis. Cover: shell-quoting of multi-line GraphQL mutations, bodyMod.replace exact-match brittleness, hallucinated --body-replace-old/--body-replace-new flags on beans update, BeanFilter field-name guessing (parentId vs parent), and not guessing bean IDs for beans show. Open PR against upstream/main.

## Summary of Changes

Added a 'Common failure modes when invoked by agents' section to CLAUDE.md (54 lines, ~75-line addition with code blocks). Covers the five clusters from the failure analysis with concrete BAD/GOOD examples:

- Multi-line GraphQL mutations under shell quoting (clusters 1, 2, 4, 7, 12)
- bodyMod.replace exact-match brittleness (cluster 3)
- Hallucinated --body-replace-old/--body-replace-new flags on beans update (cluster 11)
- BeanFilter parentId vs parent, String vs [String] (clusters 9, 10)
- beans show errors on unknown IDs (clusters 5, 6)

Section placed before the # Testing section. PR opened against upstream hmans/beans.
