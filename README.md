# mm-guest-audit — Mattermost Guest User Auditor

## What It Does

`mm-guest-audit` produces a comprehensive audit report of all guest users on a Mattermost instance. For each guest, it shows which teams and channels they can access, when they last logged in and posted, and whether their account is active or deactivated. You can optionally flag guests who have been inactive beyond a configurable threshold — useful for periodic access reviews.

## Why You'd Use It

Guest accounts represent external users — contractors, partners, clients — with access to specific teams and channels on your Mattermost instance. There is no consolidated view in the Mattermost UI or `mmctl` that shows all guests alongside their access scope and activity.

If you need to produce evidence for a security audit (ISO 27001, Cyber Essentials, SOC 2), conduct a periodic access review, or identify stale guest accounts for off-boarding, this tool gives you everything in a single command. Output can be exported as CSV for spreadsheets, JSON for programmatic processing, or a human-readable table for quick review.

## Installation

Download the pre-built binary for your platform from the [Releases](https://github.com/jlandells/mm-guest-audit/releases) page.

| Platform              | Filename                             |
|-----------------------|--------------------------------------|
| Linux (amd64)         | `mm-guest-audit-linux-amd64`         |
| macOS (Intel)         | `mm-guest-audit-darwin-amd64`        |
| macOS (Apple Silicon) | `mm-guest-audit-darwin-arm64`        |
| Windows (amd64)       | `mm-guest-audit-windows-amd64.exe`   |

On Linux and macOS, make the binary executable after downloading:

```bash
chmod +x mm-guest-audit-*
```

No other dependencies or installation steps are required.

## Authentication

The tool requires a System Administrator account to access the necessary API endpoints.

### Personal Access Token (Recommended)

Generate a Personal Access Token in **System Console > Integrations > Integration Management** (or from your profile settings if permitted by your admin). Pass it via `--token` or the `MM_TOKEN` environment variable.

```bash
mm-guest-audit --url https://mattermost.example.com --token your-token-here
```

### Username and Password

If Personal Access Tokens are disabled on your instance, you can authenticate with a username and password. The tool will prompt you for the password interactively (with echo suppressed):

```bash
mm-guest-audit --url https://mattermost.example.com --username admin
Password:
```

For non-interactive use (scripts, CI/CD), set the `MM_PASSWORD` environment variable:

```bash
export MM_URL=https://mattermost.example.com
export MM_USERNAME=admin
export MM_PASSWORD=your-password
mm-guest-audit
```

**Note:** There is no `--password` flag. Passwords passed as CLI arguments appear in shell history and process listings, which is a security risk.

## Usage

```
mm-guest-audit [flags]
```

### Flag Reference

| Flag | Env Var | Type | Default | Description |
|------|---------|------|---------|-------------|
| `--url` | `MM_URL` | string | *(required)* | Mattermost server URL |
| `--token` | `MM_TOKEN` | string | | Personal Access Token |
| `--username` | `MM_USERNAME` | string | | Username for password auth |
| `--team` | | string | *(all teams)* | Scope report to a single named team |
| `--inactive-days` | | int | `0` (disabled) | Flag guests inactive for more than N days |
| `--format` | | string | `table` | Output format: `table`, `csv`, `json` |
| `--output` | | string | *(stdout)* | Write output to a file |
| `--verbose` / `-v` | | bool | `false` | Enable verbose logging to stderr |
| `--version` | | bool | `false` | Print version and exit |

## Examples

### Basic run with token authentication

```bash
mm-guest-audit --url https://mattermost.example.com --token xoxb-your-token
```

### Basic run with username/password authentication

```bash
mm-guest-audit --url https://mattermost.example.com --username admin
Password:
```

### Using environment variables

```bash
export MM_URL=https://mattermost.example.com
export MM_TOKEN=xoxb-your-token
mm-guest-audit
```

### Write CSV report to a file

```bash
mm-guest-audit --url https://mattermost.example.com --token TOKEN --format csv --output guest-report.csv
```

### Flag inactive guests (no login in 30 days)

```bash
mm-guest-audit --url https://mattermost.example.com --token TOKEN --inactive-days 30
```

### Scope to a single team

```bash
mm-guest-audit --url https://mattermost.example.com --token TOKEN --team Engineering
```

### JSON output for scripting

```bash
mm-guest-audit --url https://mattermost.example.com --token TOKEN --format json | jq '.guests[] | select(.inactive == true)'
```

## Output Formats

### Table (default)

Human-readable tabular output. Long channel lists are truncated. A summary line is printed at the end.

```
USERNAME        DISPLAY NAME     EMAIL                      TEAMS          CHANNELS                       LAST LOGIN        LAST POST         STATUS
jane.doe        Jane Doe         jane.doe@external.com      Engineering    General, Dev Backend (+1 more)  2024-11-15 08:32  2024-11-14 17:22  Active
bob.contractor  Bob Contractor   bob@contractor.io          Engineering    General                         Never             Never             Inactive

Total: 2 guest(s) — 1 active, 1 inactive
```

### CSV

One row per guest. Multi-value fields use pipe (`|`) separators. Dates in ISO 8601 format.

```csv
username,display_name,email,created_at,last_login,last_post,teams,channels,active,inactive
jane.doe,Jane Doe,jane.doe@external.com,2024-03-01T10:00:00Z,2024-11-15T08:32:00Z,2024-11-14T17:22:00Z,Engineering|Sales,Engineering/General|Engineering/Dev Backend|Sales/Partner Updates,true,false
bob.contractor,Bob Contractor,bob@contractor.io,2024-03-01T10:00:00Z,,,,Engineering,Engineering/General,true,true
```

### JSON

Structured JSON with a top-level `summary` object and a `guests` array. Null dates are represented as JSON `null`.

```json
{
  "summary": {
    "total_guests": 2,
    "active_guests": 1,
    "inactive_guests": 1,
    "deactivated_guests": 0,
    "failed_lookups": 0
  },
  "inactive_days": 30,
  "guests": [
    {
      "username": "jane.doe",
      "display_name": "Jane Doe",
      "email": "jane.doe@external.com",
      "created_at": "2024-03-01T10:00:00Z",
      "last_login": "2024-11-15T08:32:00Z",
      "last_post": "2024-11-14T17:22:00Z",
      "teams": ["Engineering", "Sales"],
      "channels": [
        { "team": "Engineering", "channel": "General" },
        { "team": "Engineering", "channel": "Dev Backend" },
        { "team": "Sales", "channel": "Partner Updates" }
      ],
      "active": true,
      "inactive": false
    },
    {
      "username": "bob.contractor",
      "display_name": "Bob Contractor",
      "email": "bob@contractor.io",
      "created_at": "2024-03-01T10:00:00Z",
      "last_login": null,
      "last_post": null,
      "teams": ["Engineering"],
      "channels": [
        { "team": "Engineering", "channel": "General" }
      ],
      "active": true,
      "inactive": true
    }
  ]
}
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success — report generated |
| `1` | Configuration error — missing URL, invalid auth, unknown team name |
| `2` | API error — connection failure, unexpected server response |
| `3` | Partial failure — report generated but some guest lookups failed |
| `4` | Output error — unable to write to the specified output file |

## Limitations

- **Last post date uses search** — the Mattermost API does not expose a "last post date" field on user objects. This tool retrieves it by searching for posts by each guest in each of their teams. On large instances with many guests and teams, this can result in a significant number of API calls and may be slow. If `--team` is specified, only that team is searched, which significantly reduces the number of calls.
- **Rate limiting** — on very large instances, the volume of API calls (one per guest per team for channels, plus search queries for last post dates) may approach rate limits. If you encounter rate limiting errors, try scoping to a single team with `--team`.
- **Read-only** — this tool does not deactivate, remove, or modify guest accounts in any way. It is a reporting tool only.

## Integration Testing

To test against a local Mattermost instance:

1. Run a local Mattermost server (e.g. via Docker)
2. Create a System Administrator account and generate a Personal Access Token
3. Create one or more guest accounts with team and channel memberships
4. Run the tool against the local instance:
   ```bash
   mm-guest-audit --url http://localhost:8065 --token your-local-token --verbose
   ```
5. Verify the output matches the guest accounts you created

## Contributing

We welcome contributions from the community! Whether it's a bug report, a feature suggestion,
or a pull request, your input is valuable to us. Please feel free to contribute in the
following ways:
- **Issues and Pull Requests**: For specific questions, issues, or suggestions for improvements,
  open an issue or a pull request in this repository.
- **Mattermost Community**: Join the discussion in the
  [Integrations and Apps](https://community.mattermost.com/core/channels/integrations) channel
  on the Mattermost Community server.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contact

For questions, feedback, or contributions regarding this project, please use the following methods:
- **Issues and Pull Requests**: For specific questions, issues, or suggestions for improvements,
  feel free to open an issue or a pull request in this repository.
- **Mattermost Community**: Join us in the Mattermost Community server, where we discuss all
  things related to extending Mattermost. You can find me in the channel
  [Integrations and Apps](https://community.mattermost.com/core/channels/integrations).
- **Social Media**: Follow and message me on Twitter, where I'm
  [@jlandells](https://twitter.com/jlandells).
