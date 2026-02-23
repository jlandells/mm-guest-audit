# PRD: mm-guest-audit — Mattermost Guest User Auditor

**Version:** 1.0  
**Status:** Ready for Development  
**Language:** Go  
**Binary Name:** `mm-guest-audit`

---

## 1. Overview

`mm-guest-audit` is a standalone command-line utility that produces a comprehensive audit report of all guest users on a Mattermost instance. It reports which teams and channels each guest can access, when they last logged in and posted, and flags guests who are inactive beyond a configurable threshold. Output can be written to stdout or a file in table, CSV, or JSON format.

---

## 2. Background & Problem Statement

Mattermost supports guest accounts — external users with restricted access, scoped to specific teams and channels. There is no consolidated view in the Mattermost UI or `mmctl` that shows all guests across an instance alongside their access scope and activity. Administrators must click through multiple screens to gather this information manually.

In regulated and enterprise environments, this is a genuine problem. Guest accounts represent external parties — contractors, partners, clients — who have access to potentially sensitive channels. Security audits, access reviews (e.g. for ISO 27001, Cyber Essentials, or internal governance requirements), and periodic off-boarding processes all require this information quickly and reliably.

This tool makes what is currently a manual, error-prone process into a single command.

---

## 3. Goals

- Provide a complete, accurate list of all guest users on a Mattermost instance
- For each guest, show their team and channel access, and their activity dates
- Allow admins to filter by inactivity threshold to identify guests who may no longer need access
- Produce output suitable for inclusion in audit reports (CSV, JSON) or for human review (table)
- Be trivial to run with no installation beyond downloading the binary

---

## 4. Non-Goals

- This tool does not deactivate, remove, or modify any guest users or their access — it is read-only
- This tool does not send notifications or alerts
- This tool does not integrate with any external identity provider or directory service
- This tool does not manage guest invitations

---

## 5. Target Users

Mattermost System Administrators, particularly those in regulated industries (defence, government, financial services, healthcare) who need to produce evidence of access governance.

---

## 6. User Stories

- As a System Administrator, I want to see a full list of all guest accounts on my instance so that I can review who has external access.
- As a System Administrator, I want to see which teams and channels each guest can access so that I can verify their access is still appropriate.
- As a System Administrator, I want to flag guests who have not logged in or posted for 30 days so that I can raise them for review or removal.
- As a Security Officer, I want a CSV export of all guest access so that I can include it in our quarterly access review documentation.
- As a System Administrator, I want to scope the report to a single team so that individual team owners can review their own guests.

---

## 7. Functional Requirements

### 7.1 Guest Enumeration

- The tool MUST retrieve all users whose Mattermost role is `system_guest`
- The tool MUST paginate through all results — it MUST NOT assume all users fit in a single API response
- For each guest user, the tool MUST retrieve and report:
  - Username
  - Display name (first name + last name)
  - Email address
  - Account creation date (ISO 8601)
  - Last login date (ISO 8601, or "Never" if null)
  - Last post date (ISO 8601, or "Never" if null) — see note in section 7.2
  - List of teams the guest is a member of (by team display name)
  - List of channels the guest has access to (by channel display name, grouped by team)
  - Whether the guest account is currently active or deactivated

### 7.2 Last Post Date

The Mattermost API does not expose a direct "last post date" field on the user object. The tool should retrieve this by querying the user's post history. If this is not feasible within acceptable API call limits for large instances, the field should be omitted and documented as a known limitation in the README, rather than silently showing incorrect data.

### 7.3 Inactivity Flagging

- When `--inactive-days N` is specified, the tool MUST flag any guest whose last login date is more than N days ago (or who has never logged in)
- Flagged guests should be visually distinguished in table output (e.g. with an `[INACTIVE]` marker or equivalent)
- In CSV and JSON output, an `inactive` boolean field should be included

### 7.4 Team Scoping

- When `--team TEAM_NAME` is specified, the tool MUST resolve the team name to a team ID via the API
- If the team name cannot be resolved, the tool MUST exit with a clear error message — it MUST NOT silently return empty results
- When scoped, only guests who are members of that team should be included in the report

### 7.5 Output

See Section 9 (Output Specification).

---

## 8. CLI Specification

### Usage

```
mm-guest-audit [flags]
```

### Connection Flags (required)

| Flag | Environment Variable | Description |
|------|----------------------|-------------|
| `--url URL` | `MM_URL` | Mattermost server URL, e.g. `https://mattermost.example.com` |

### Authentication Flags

| Flag | Environment Variable | Description |
|------|----------------------|-------------|
| `--token TOKEN` | `MM_TOKEN` | Personal Access Token (preferred) |
| `--username USERNAME` | `MM_USERNAME` | Username for password-based auth |
| *(no flag)* | `MM_PASSWORD` | Password (env var only — never a CLI flag) |

Authentication resolution order:
1. `--token` / `MM_TOKEN`
2. `--username` + interactive password prompt (if terminal is interactive)
3. `--username` + `MM_PASSWORD` environment variable

### Operational Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--team TEAM_NAME` | *(none — all teams)* | Scope report to a single named team |
| `--inactive-days N` | *(no flagging)* | Flag guests with no activity in the last N days |
| `--format table\|csv\|json` | `table` | Output format |
| `--output FILE` | *(stdout)* | Write output to a file |
| `--verbose` / `-v` | `false` | Enable verbose logging to stderr |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Configuration error (missing URL, invalid auth, unknown team name) |
| `2` | API error (connection failure, unexpected response) |
| `3` | Output error (unable to write file) |

---

## 9. Output Specification

### 9.1 Table Format

Human-readable tabular output to stdout. Columns:

```
USERNAME | DISPLAY NAME | EMAIL | TEAMS | CHANNELS | LAST LOGIN | LAST POST | STATUS
```

Where STATUS is `Active`, `Deactivated`, or `Inactive` (the last being used when `--inactive-days` is set and the threshold is exceeded).

Long channel lists should be truncated in table view with a count suffix, e.g. `general, townhall (+4 more)`.

### 9.2 CSV Format

One row per guest. Columns:

```
username, display_name, email, created_at, last_login, last_post, teams, channels, active, inactive
```

- `teams` — pipe-separated list of team display names, e.g. `Engineering|Sales`
- `channels` — pipe-separated list of `team/channel` pairs, e.g. `Engineering/general|Engineering/dev-backend`
- `active` — `true` or `false`
- `inactive` — `true` or `false` (only meaningful when `--inactive-days` is set; otherwise always `false`)

### 9.3 JSON Format

Array of guest objects:

```json
[
  {
    "username": "jane.doe",
    "display_name": "Jane Doe",
    "email": "jane.doe@external.com",
    "created_at": "2024-03-01T10:00:00Z",
    "last_login": "2024-11-15T08:32:00Z",
    "last_post": "2024-11-14T17:22:00Z",
    "teams": ["Engineering", "Sales"],
    "channels": [
      { "team": "Engineering", "channel": "general" },
      { "team": "Sales", "channel": "partner-updates" }
    ],
    "active": true,
    "inactive": false
  }
]
```

---

## 10. Authentication Detail

### Personal Access Token

The token must belong to a user with the **System Administrator** role. A token belonging to a regular user or guest will not have the permissions required to enumerate all users and team/channel memberships.

### Username and Password

- The username must belong to a System Administrator account
- If the tool detects it is running in an interactive terminal (i.e. stdin is a TTY), it MUST prompt for the password with echo suppressed using `golang.org/x/term`
- If running non-interactively and `MM_PASSWORD` is set, that value should be used
- If running non-interactively and `MM_PASSWORD` is not set, the tool MUST exit with a clear error message rather than hanging waiting for input

---

## 11. API Endpoints Used

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v4/users?role=system_guest&page=N&per_page=200` | Enumerate all guest users (paginated) |
| `GET /api/v4/teams/name/{team_name}` | Resolve team name to ID |
| `GET /api/v4/users/{user_id}/teams` | Get teams for a specific guest |
| `GET /api/v4/users/{user_id}/teams/{team_id}/channels` | Get channels for a guest within a team |

All requests must include the `Authorization` header — either `Bearer {token}` for PAT auth, or a session token obtained by `POST /api/v4/users/login` for username/password auth.

---

## 12. Error Handling

- If `--url` is not provided (and `MM_URL` is not set), exit immediately with: `Error: server URL is required. Use --url or set MM_URL.`
- If authentication fails (401 response), exit with: `Error: authentication failed. Check your token or credentials.`
- If the specified team name cannot be resolved, exit with: `Error: team "TEAM_NAME" not found. Check the name and try again.`
- If the API returns an unexpected error, print the HTTP status and response body to stderr and exit with code 2
- Individual guest lookup failures (e.g. a 404 mid-pagation) should be logged to stderr as warnings and skipped, rather than aborting the entire run — the tool should complete and note the number of skipped records in the summary

---

## 13. Testing Requirements

The developer should provide:

- Unit tests for the inactivity calculation logic
- Unit tests for name-to-ID resolution (with mocked API responses)
- Unit tests for CSV and JSON output formatting
- Integration test instructions in the README covering a real Mattermost instance (can be a local Docker instance for development purposes)

---

## 14. Out of Scope

- Deactivating or modifying guest accounts
- Sending reports by email or posting them to a Mattermost channel
- Scheduling or running on a recurring basis
- Comparing guest access against an expected baseline

---

## 15. Acceptance Criteria

- [ ] Running `mm-guest-audit --url https://mm.example.com --token TOKEN` produces a table of all guests
- [ ] `--team Engineering` correctly scopes results to guests in the Engineering team only
- [ ] `--inactive-days 30` correctly flags guests with no login in the last 30 days
- [ ] `--format csv --output report.csv` produces a valid, importable CSV file
- [ ] `--format json` produces valid JSON that can be parsed by `jq`
- [ ] Running without `--url` or `MM_URL` produces a clear error and exits with code 1
- [ ] Running with an invalid token produces a clear error and exits with code 1
- [ ] Running with `--team NonExistentTeam` produces a clear error and exits with code 1
- [ ] All errors go to stderr; all data output goes to stdout
- [ ] The binary runs on Linux (amd64), macOS (arm64 and amd64), and Windows (amd64) without any dependencies