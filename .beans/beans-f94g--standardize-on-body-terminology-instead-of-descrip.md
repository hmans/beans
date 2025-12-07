---
title: Standardize on 'description' terminology instead of 'body'
status: done
created_at: 2025-12-06T23:30:26Z
updated_at: 2025-12-06T23:40:48Z
---



Standardized terminology to use 'description' everywhere instead of 'body'.

## Changes made
- [x] Renamed `Body` field to `Description` in Bean struct (`internal/bean/bean.go`)
- [x] Changed JSON tag from `body` to `description`
- [x] Renamed `--body-only` flag to `--description-only` in show command
- [x] Renamed `--include-body` flag to `--include-description` in list command
- [x] Updated all variable names and comments
- [x] Updated help text
- [x] Updated tests