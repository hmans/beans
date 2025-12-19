---
title: Use semantic markdown sections in bean bodies
status: draft
type: feature
tags:
  - parsing
  - schema
  - idea
created_at: 2025-12-07T12:33:57Z
updated_at: 2025-12-08T17:02:44Z
---

## Overview

Establish a convention for using markdown headings in bean bodies to create semantically meaningful sections. This would be guided by the beans system prompt rather than enforced programmatically.

## Proposed Sections

### Issue Description

The main description of what needs to be done or what the issue is about.

### Changelog Content

What should appear in the changelog when this bean is released. Should include a link back to the issue (e.g., `(#beans-xyz)`).

### Open Questions

Uncertainties, decisions to be made, or things that need clarification.

### History

A bullet-point list of changes/updates made to the issue over time, providing context for how the issue evolved.

## Implementation

- This is primarily a **convention**, not a code change
- Update `prompt.md` (the beans system prompt) to guide agents toward using these sections
- Agents should:
  - Add to History when making significant updates
  - Fill in Changelog Content when completing work
  - Use Open Questions to track uncertainties

## Relationship to Release Management

This pairs well with the release management idea (beans-5p12) - the Changelog Content section would provide the text used when generating CHANGELOG.md entries.

## Checklist

- [ ] Define the canonical section names and their purposes
- [ ] Update prompt.md to instruct agents on using these sections
- [ ] Consider: Should some sections be optional vs. recommended?
- [ ] Consider: Template for new beans with section stubs?
