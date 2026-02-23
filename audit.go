package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// TeamInfo represents a team a guest belongs to.
type TeamInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// ChannelInfo represents a channel a guest can access.
type ChannelInfo struct {
	TeamName    string `json:"team"`
	ChannelName string `json:"channel"`
}

// GuestRecord holds all audit information for a single guest user.
type GuestRecord struct {
	Username    string        `json:"username"`
	DisplayName string        `json:"display_name"`
	Email       string        `json:"email"`
	CreatedAt   *time.Time    `json:"created_at"`
	LastLogin   *time.Time    `json:"last_login"`
	LastPost    *time.Time    `json:"last_post"`
	Teams       []TeamInfo    `json:"teams"`
	Channels    []ChannelInfo `json:"channels"`
	Active      bool          `json:"active"`
	Inactive    bool          `json:"inactive"`
	Error       string        `json:"error,omitempty"`
}

// AuditSummary holds aggregate counts for the audit.
type AuditSummary struct {
	TotalGuests       int `json:"total_guests"`
	ActiveGuests      int `json:"active_guests"`
	InactiveGuests    int `json:"inactive_guests"`
	DeactivatedGuests int `json:"deactivated_guests"`
	FailedLookups     int `json:"failed_lookups"`
}

// AuditResult holds the complete audit output.
type AuditResult struct {
	Guests       []GuestRecord `json:"guests"`
	Summary      AuditSummary  `json:"summary"`
	InactiveDays int           `json:"inactive_days"`
}

// RunAudit performs the guest audit against the Mattermost instance.
func RunAudit(client MattermostClient, teamFilter string, inactiveDays int, verbose bool) (*AuditResult, int) {
	var filterTeamID string
	var filterTeamName string

	// Resolve team filter if set
	if teamFilter != "" {
		team, err := client.GetTeamByName(teamFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return nil, ExitConfigError
		}
		filterTeamID = team.Id
		filterTeamName = team.DisplayName
		if verbose {
			fmt.Fprintf(os.Stderr, "Scoping to team: %s (ID: %s)\n", filterTeamName, filterTeamID)
		}
	}

	// Paginate through all guest users
	if verbose {
		fmt.Fprintln(os.Stderr, "Retrieving guest users...")
	}
	var allGuests []*model.User
	page := 0
	perPage := 200
	for {
		users, err := client.GetGuestUsers(page, perPage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return nil, ExitAPIError
		}
		allGuests = append(allGuests, users...)
		if len(users) < perPage {
			break
		}
		page++
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d guest user(s)\n", len(allGuests))
	}

	// Process each guest
	result := &AuditResult{
		InactiveDays: inactiveDays,
	}
	exitCode := ExitSuccess

	for _, u := range allGuests {
		record, err := processGuest(client, u, filterTeamID, inactiveDays, verbose)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to process guest %q: %v\n", u.Username, err)
			}
			record = &GuestRecord{
				Username:    u.Username,
				DisplayName: BuildDisplayName(u.FirstName, u.LastName),
				Email:       u.Email,
				CreatedAt:   MillisToTime(u.CreateAt),
				Active:      u.DeleteAt == 0,
				Error:       err.Error(),
			}
			result.Summary.FailedLookups++
			exitCode = ExitPartialFailure
		}

		// Skip guests not in the filtered team (processGuest returns nil)
		if record == nil {
			continue
		}

		result.Guests = append(result.Guests, *record)
	}

	// Calculate summary
	for _, g := range result.Guests {
		if g.Error != "" {
			continue
		}
		if !g.Active {
			result.Summary.DeactivatedGuests++
		} else if g.Inactive {
			result.Summary.InactiveGuests++
		} else {
			result.Summary.ActiveGuests++
		}
	}
	result.Summary.TotalGuests = len(result.Guests)

	return result, exitCode
}

// processGuest enriches a single guest user with team, channel, and activity data.
func processGuest(client MattermostClient, u *model.User, filterTeamID string, inactiveDays int, verbose bool) (*GuestRecord, error) {
	// Get teams for this user
	teams, err := client.GetTeamsForUser(u.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}

	// Filter teams if team scoping is active
	var teamInfos []TeamInfo
	for _, t := range teams {
		if filterTeamID != "" && t.Id != filterTeamID {
			continue
		}
		teamInfos = append(teamInfos, TeamInfo{
			ID:          t.Id,
			DisplayName: t.DisplayName,
		})
	}

	// If team filter is active and this guest is not in that team, skip
	if filterTeamID != "" && len(teamInfos) == 0 {
		return nil, nil
	}

	// Get channels per team
	var channels []ChannelInfo
	var teamIDs []string
	for _, ti := range teamInfos {
		teamIDs = append(teamIDs, ti.ID)
		chs, err := client.GetChannelsForTeamForUser(ti.ID, u.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to get channels for team %q: %w", ti.DisplayName, err)
		}
		for _, ch := range chs {
			channels = append(channels, ChannelInfo{
				TeamName:    ti.DisplayName,
				ChannelName: ch.DisplayName,
			})
		}
	}

	// Get last post date
	var lastPost *time.Time
	if len(teamIDs) > 0 {
		lastPost, err = client.GetLastPostDateForUser(u.Id, u.Username, teamIDs)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: could not retrieve last post date for %q: %v\n", u.Username, err)
			}
			// Non-fatal — continue without last post date
		}
	}

	lastLogin := MillisToTime(u.LastActivityAt)
	active := u.DeleteAt == 0
	inactive := IsInactive(lastLogin, inactiveDays)

	record := &GuestRecord{
		Username:    u.Username,
		DisplayName: BuildDisplayName(u.FirstName, u.LastName),
		Email:       u.Email,
		CreatedAt:   MillisToTime(u.CreateAt),
		LastLogin:   lastLogin,
		LastPost:    lastPost,
		Teams:       teamInfos,
		Channels:    channels,
		Active:      active,
		Inactive:    inactive,
	}

	return record, nil
}

// IsInactive determines whether a guest should be flagged as inactive.
// A guest is inactive if inactiveDays > 0 and their last login is more than
// inactiveDays ago (or they have never logged in).
func IsInactive(lastLogin *time.Time, inactiveDays int) bool {
	if inactiveDays <= 0 {
		return false
	}
	if lastLogin == nil {
		return true // never logged in
	}
	cutoff := time.Now().AddDate(0, 0, -inactiveDays)
	return lastLogin.Before(cutoff)
}

// IsInactiveAt is a testable version of IsInactive that accepts a reference time.
func IsInactiveAt(lastLogin *time.Time, inactiveDays int, now time.Time) bool {
	if inactiveDays <= 0 {
		return false
	}
	if lastLogin == nil {
		return true
	}
	cutoff := now.AddDate(0, 0, -inactiveDays)
	return lastLogin.Before(cutoff)
}

// BuildDisplayName combines first and last name into a display name.
func BuildDisplayName(firstName, lastName string) string {
	switch {
	case firstName != "" && lastName != "":
		return firstName + " " + lastName
	case firstName != "":
		return firstName
	case lastName != "":
		return lastName
	default:
		return ""
	}
}

// MillisToTime converts a Unix millisecond timestamp to *time.Time.
// Returns nil if the timestamp is zero (representing no date).
func MillisToTime(millis int64) *time.Time {
	if millis == 0 {
		return nil
	}
	t := time.UnixMilli(millis).UTC()
	return &t
}

// FormatTimeISO formats a *time.Time as ISO 8601, or returns an empty string if nil.
func FormatTimeISO(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// FormatTimeDisplay formats a *time.Time for table display, or returns "Never" if nil.
func FormatTimeDisplay(t *time.Time) string {
	if t == nil {
		return "Never"
	}
	return t.UTC().Format("2006-01-02 15:04")
}
