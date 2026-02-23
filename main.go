package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	// Connection flags
	url := flag.String("url", envOrDefault("MM_URL", ""), "Mattermost server URL")
	token := flag.String("token", envOrDefault("MM_TOKEN", ""), "Personal Access Token")
	username := flag.String("username", envOrDefault("MM_USERNAME", ""), "Username for password auth")

	// Operational flags
	team := flag.String("team", "", "Scope report to a single named team")
	inactiveDays := flag.Int("inactive-days", 0, "Flag guests with no activity in the last N days")
	format := flag.String("format", "table", "Output format: table, csv, json")
	output := flag.String("output", "", "Write output to this file path")
	verbose := flag.Bool("verbose", false, "Enable verbose logging to stderr")
	showVersion := flag.Bool("version", false, "Print version and exit")

	// Short flag aliases
	flag.BoolVar(verbose, "v", false, "Enable verbose logging to stderr")

	flag.Parse()

	if *showVersion {
		fmt.Printf("mm-guest-audit %s\n", version)
		return ExitSuccess
	}

	// Validate URL
	if *url == "" {
		fmt.Fprintln(os.Stderr, "error: server URL is required. Use --url or set the MM_URL environment variable.")
		return ExitConfigError
	}

	// Validate format
	switch *format {
	case "table", "csv", "json":
		// valid
	default:
		fmt.Fprintf(os.Stderr, "error: invalid format %q. Use table, csv, or json.\n", *format)
		return ExitConfigError
	}

	// Authenticate
	client, err := NewClient(*url, *token, *username, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return ExitConfigError
	}

	if *verbose {
		fmt.Fprintln(os.Stderr, "Authentication successful.")
	}

	// Run audit
	result, exitCode := RunAudit(client, *team, *inactiveDays, *verbose)
	if result == nil {
		return exitCode
	}

	// Write output
	if err := WriteOutput(result, *format, *output); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to write output: %v\n", err)
		return ExitOutputError
	}

	return exitCode
}

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
