# Architecture — mm-guest-audit

This document describes the design decisions and architecture of the `mm-guest-audit` tool.

## Overview

`mm-guest-audit` is a read-only CLI tool that audits all guest users on a Mattermost instance. It follows the conventions defined in `CLAUDE.md` for the Mattermost Admin Utilities family.

## File Layout

| File | Responsibility |
|------|---------------|
| `main.go` | Entry point — flag parsing, validation, orchestration. No business logic. |
| `client.go` | `MattermostClient` interface and its real implementation wrapping `model.Client4`. |
| `audit.go` | Core business logic — guest enumeration, team/channel resolution, inactivity calculation. |
| `output.go` | Output formatters for table, CSV, and JSON. File writer with stdout fallback. |
| `errors.go` | Exit code constants. |

## Key Design Decisions

### Interface-Based Client

All Mattermost API interactions go through the `MattermostClient` interface defined in `client.go`. This enables:

- **Unit testing without a Mattermost instance** — tests use a mock implementation
- **Clear API boundary** — the interface documents exactly which API calls the tool makes
- **Future flexibility** — the implementation could be swapped without changing business logic

### Exit Code Mapping

The PRD defines exit code 3 as "output error". CLAUDE.md (the family standard) defines exit code 3 as "partial failure" and 4 as "output error". We follow **CLAUDE.md** since it is the authoritative cross-tool standard:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Configuration error |
| 2 | API error |
| 3 | Partial failure |
| 4 | Output error |

### Last Post Date Strategy

The Mattermost API does not expose a "last post date" field on the user object. We use `SearchPosts` with a `from:{username}` query per team:

- When `--team` is set, only that team is searched (significantly reduces API calls)
- Without `--team`, all teams the guest belongs to are searched
- The most recent `CreateAt` across all matching posts is used
- This can be slow on large instances with many teams — documented in the README as a known limitation

If last post date retrieval fails for a specific guest, it is treated as non-fatal — the guest record is still included with a nil last post date.

### Pagination

All API calls that return lists are paginated with `per_page=200` (the Mattermost maximum). The pagination loop continues until a page returns fewer than `per_page` results.

### Partial Failures

When processing fails for an individual guest (e.g. team lookup returns a 500), the tool:

1. Records the guest with an error message in the output
2. Logs the error to stderr if `--verbose` is active
3. Continues processing remaining guests
4. Returns exit code 3 (partial failure) instead of 0

This ensures that one problematic guest account does not prevent the audit of all others.

### Output File Fallback

If the `--output` file cannot be opened for writing, the tool:

1. Prints a warning to stderr
2. Falls back to writing to stdout
3. Does NOT exit with an error code in this case — the data is still delivered

### Password Handling

In accordance with CLAUDE.md:

- No `--password` flag exists
- Interactive TTY sessions prompt for the password with echo suppressed (`golang.org/x/term`)
- Non-interactive sessions read from `MM_PASSWORD` environment variable
- If neither is available, the tool exits with a clear error

## Data Flow

```
main.go
  ├── Parse flags, validate input
  ├── NewClient() → authenticate
  ├── RunAudit()
  │     ├── Resolve --team filter (if set)
  │     ├── Paginate all guest users
  │     └── Per guest:
  │           ├── GetTeamsForUser()
  │           ├── Filter by team (if scoped)
  │           ├── GetChannelsForTeamForUser() per team
  │           ├── GetLastPostDateForUser()
  │           └── Calculate inactivity
  └── WriteOutput() → table/csv/json to file/stdout
```
