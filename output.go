package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// WriteOutput writes the audit result in the specified format to the specified destination.
func WriteOutput(result *AuditResult, format, outputPath string) error {
	var w io.Writer = os.Stdout

	if outputPath != "" {
		f, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: unable to write to %q: %v — writing to stdout instead\n", outputPath, err)
		} else {
			defer f.Close()
			w = f
		}
	}

	switch format {
	case "csv":
		return writeCSV(w, result)
	case "json":
		return writeJSON(w, result)
	default:
		return writeTable(w, result)
	}
}

func writeTable(w io.Writer, result *AuditResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, "USERNAME\tDISPLAY NAME\tEMAIL\tTEAMS\tCHANNELS\tLAST LOGIN\tLAST POST\tSTATUS")

	for _, g := range result.Guests {
		teams := formatTeamNames(g.Teams)
		channels := formatChannelNamesTable(g.Channels)
		status := guestStatus(g)

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			g.Username,
			g.DisplayName,
			g.Email,
			teams,
			channels,
			FormatTimeDisplay(g.LastLogin),
			FormatTimeDisplay(g.LastPost),
			status,
		)
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	// Summary
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Total: %d guest(s)", result.Summary.TotalGuests)
	parts := []string{}
	if result.Summary.ActiveGuests > 0 {
		parts = append(parts, fmt.Sprintf("%d active", result.Summary.ActiveGuests))
	}
	if result.Summary.InactiveGuests > 0 {
		parts = append(parts, fmt.Sprintf("%d inactive", result.Summary.InactiveGuests))
	}
	if result.Summary.DeactivatedGuests > 0 {
		parts = append(parts, fmt.Sprintf("%d deactivated", result.Summary.DeactivatedGuests))
	}
	if result.Summary.FailedLookups > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", result.Summary.FailedLookups))
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, " — %s", strings.Join(parts, ", "))
	}
	fmt.Fprintln(w)

	return nil
}

func writeCSV(w io.Writer, result *AuditResult) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header row
	header := []string{"username", "display_name", "email", "created_at", "last_login", "last_post", "teams", "channels", "active", "inactive"}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, g := range result.Guests {
		row := []string{
			g.Username,
			g.DisplayName,
			g.Email,
			FormatTimeISO(g.CreatedAt),
			FormatTimeISO(g.LastLogin),
			FormatTimeISO(g.LastPost),
			formatTeamNamesCSV(g.Teams),
			formatChannelNamesCSV(g.Channels),
			fmt.Sprintf("%t", g.Active),
			fmt.Sprintf("%t", g.Inactive),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// jsonOutput is the top-level JSON structure for output.
type jsonOutput struct {
	Summary      AuditSummary      `json:"summary"`
	InactiveDays int               `json:"inactive_days"`
	Guests       []jsonGuestRecord `json:"guests"`
}

// jsonGuestRecord is the JSON representation of a guest, with nullable date fields.
type jsonGuestRecord struct {
	Username    string        `json:"username"`
	DisplayName string        `json:"display_name"`
	Email       string        `json:"email"`
	CreatedAt   *string       `json:"created_at"`
	LastLogin   *string       `json:"last_login"`
	LastPost    *string       `json:"last_post"`
	Teams       []string      `json:"teams"`
	Channels    []ChannelInfo `json:"channels"`
	Active      bool          `json:"active"`
	Inactive    bool          `json:"inactive"`
}

func writeJSON(w io.Writer, result *AuditResult) error {
	output := jsonOutput{
		Summary:      result.Summary,
		InactiveDays: result.InactiveDays,
		Guests:       make([]jsonGuestRecord, 0, len(result.Guests)),
	}

	for _, g := range result.Guests {
		teamNames := make([]string, 0, len(g.Teams))
		for _, t := range g.Teams {
			teamNames = append(teamNames, t.DisplayName)
		}

		channels := g.Channels
		if channels == nil {
			channels = []ChannelInfo{}
		}

		record := jsonGuestRecord{
			Username:    g.Username,
			DisplayName: g.DisplayName,
			Email:       g.Email,
			CreatedAt:   timeToStringPtr(g.CreatedAt),
			LastLogin:   timeToStringPtr(g.LastLogin),
			LastPost:    timeToStringPtr(g.LastPost),
			Teams:       teamNames,
			Channels:    channels,
			Active:      g.Active,
			Inactive:    g.Inactive,
		}
		output.Guests = append(output.Guests, record)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func timeToStringPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := FormatTimeISO(t)
	return &s
}

func guestStatus(g GuestRecord) string {
	if !g.Active {
		return "Deactivated"
	}
	if g.Inactive {
		return "Inactive"
	}
	return "Active"
}

func formatTeamNames(teams []TeamInfo) string {
	if len(teams) == 0 {
		return ""
	}
	names := make([]string, len(teams))
	for i, t := range teams {
		names[i] = t.DisplayName
	}
	return strings.Join(names, ", ")
}

func formatTeamNamesCSV(teams []TeamInfo) string {
	if len(teams) == 0 {
		return ""
	}
	names := make([]string, len(teams))
	for i, t := range teams {
		names[i] = t.DisplayName
	}
	return strings.Join(names, "|")
}

func formatChannelNamesTable(channels []ChannelInfo) string {
	if len(channels) == 0 {
		return ""
	}
	maxDisplay := 2
	names := make([]string, 0, maxDisplay)
	for i, ch := range channels {
		if i >= maxDisplay {
			break
		}
		names = append(names, ch.ChannelName)
	}
	result := strings.Join(names, ", ")
	if len(channels) > maxDisplay {
		result += fmt.Sprintf(" (+%d more)", len(channels)-maxDisplay)
	}
	return result
}

func formatChannelNamesCSV(channels []ChannelInfo) string {
	if len(channels) == 0 {
		return ""
	}
	pairs := make([]string, len(channels))
	for i, ch := range channels {
		pairs[i] = ch.TeamName + "/" + ch.ChannelName
	}
	return strings.Join(pairs, "|")
}
