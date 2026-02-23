package main

// Exit codes — consistent with the Mattermost Admin Utilities family (CLAUDE.md).
const (
	ExitSuccess        = 0 // Successful execution
	ExitConfigError    = 1 // Missing flags, invalid input, auth failure
	ExitAPIError       = 2 // Connection failure, unexpected API response
	ExitPartialFailure = 3 // Operation completed but with some failures
	ExitOutputError    = 4 // Unable to write output file
)
