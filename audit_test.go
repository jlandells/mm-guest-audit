package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// --- Mock client ---

type mockClient struct {
	guests          []*model.User
	guestsErr       error
	teams           map[string][]*model.Team // userID → teams
	teamsErr        map[string]error
	teamByName      map[string]*model.Team
	teamByNameErr   map[string]error
	channels        map[string][]*model.Channel // teamID+userID → channels
	channelsErr     map[string]error
	lastPostDate    map[string]*time.Time // userID → last post
	lastPostDateErr map[string]error
}

func (m *mockClient) GetGuestUsers(page, perPage int) ([]*model.User, error) {
	if m.guestsErr != nil {
		return nil, m.guestsErr
	}
	start := page * perPage
	if start >= len(m.guests) {
		return []*model.User{}, nil
	}
	end := start + perPage
	if end > len(m.guests) {
		end = len(m.guests)
	}
	return m.guests[start:end], nil
}

func (m *mockClient) GetTeamByName(name string) (*model.Team, error) {
	if m.teamByNameErr != nil {
		if err, ok := m.teamByNameErr[name]; ok {
			return nil, err
		}
	}
	if t, ok := m.teamByName[name]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("error: team %q not found. Please check the name and try again", name)
}

func (m *mockClient) GetTeamsForUser(userID string) ([]*model.Team, error) {
	if m.teamsErr != nil {
		if err, ok := m.teamsErr[userID]; ok {
			return nil, err
		}
	}
	return m.teams[userID], nil
}

func (m *mockClient) GetChannelsForTeamForUser(teamID, userID string) ([]*model.Channel, error) {
	key := teamID + ":" + userID
	if m.channelsErr != nil {
		if err, ok := m.channelsErr[key]; ok {
			return nil, err
		}
	}
	return m.channels[key], nil
}

func (m *mockClient) GetLastPostDateForUser(userID, username string, teamIDs []string) (*time.Time, error) {
	if m.lastPostDateErr != nil {
		if err, ok := m.lastPostDateErr[userID]; ok {
			return nil, err
		}
	}
	return m.lastPostDate[userID], nil
}

// --- Tests ---

func TestIsInactive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		lastLogin      *time.Time
		inactiveDays   int
		expectInactive bool
	}{
		{"active user, well within threshold", timePtr(now.AddDate(0, 0, -5)), 30, false},
		{"exactly at threshold boundary", timePtr(now.AddDate(0, 0, -30).Add(-time.Second)), 30, true},
		{"exactly at threshold (same moment)", timePtr(now.AddDate(0, 0, -30)), 30, false},
		{"one day over threshold", timePtr(now.AddDate(0, 0, -31)), 30, true},
		{"never logged in", nil, 30, true},
		{"inactive-days is zero (disabled)", nil, 0, false},
		{"inactive-days is zero with login", timePtr(now.AddDate(0, 0, -100)), 0, false},
		{"just under threshold", timePtr(now.AddDate(0, 0, -29)), 30, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInactiveAt(tt.lastLogin, tt.inactiveDays, now)
			if result != tt.expectInactive {
				t.Errorf("IsInactiveAt(%v, %d) = %v, want %v",
					tt.lastLogin, tt.inactiveDays, result, tt.expectInactive)
			}
		})
	}
}

func TestBuildDisplayName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		expected  string
	}{
		{"both names", "Jane", "Doe", "Jane Doe"},
		{"first only", "Jane", "", "Jane"},
		{"last only", "", "Doe", "Doe"},
		{"neither", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildDisplayName(tt.firstName, tt.lastName)
			if result != tt.expected {
				t.Errorf("BuildDisplayName(%q, %q) = %q, want %q",
					tt.firstName, tt.lastName, result, tt.expected)
			}
		})
	}
}

func TestMillisToTime(t *testing.T) {
	tests := []struct {
		name     string
		millis   int64
		expected *time.Time
	}{
		{"zero returns nil", 0, nil},
		{"valid timestamp", 1700000000000, timePtr(time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MillisToTime(tt.millis)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("MillisToTime(%d) = %v, want nil", tt.millis, result)
				}
				return
			}
			if result == nil {
				t.Fatalf("MillisToTime(%d) = nil, want %v", tt.millis, tt.expected)
			}
			if !result.Equal(*tt.expected) {
				t.Errorf("MillisToTime(%d) = %v, want %v", tt.millis, result, tt.expected)
			}
		})
	}
}

func TestFormatTimeISO(t *testing.T) {
	tests := []struct {
		name     string
		input    *time.Time
		expected string
	}{
		{"nil returns empty", nil, ""},
		{"valid time", timePtr(time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)), "2024-03-01T10:00:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeISO(tt.input)
			if result != tt.expected {
				t.Errorf("FormatTimeISO(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRunAudit_BasicScenario(t *testing.T) {
	now := time.Now()
	loginTime := now.AddDate(0, 0, -5)

	client := &mockClient{
		guests: []*model.User{
			{Id: "user1", Username: "jane.doe", FirstName: "Jane", LastName: "Doe", Email: "jane@example.com", CreateAt: 1709280000000, LastActivityAt: loginTime.UnixMilli()},
			{Id: "user2", Username: "bob.smith", FirstName: "Bob", LastName: "Smith", Email: "bob@example.com", CreateAt: 1709280000000, LastActivityAt: loginTime.UnixMilli()},
			{Id: "user3", Username: "alice.jones", FirstName: "Alice", LastName: "Jones", Email: "alice@example.com", CreateAt: 1709280000000, DeleteAt: 1710000000000},
		},
		teams: map[string][]*model.Team{
			"user1": {{Id: "team1", DisplayName: "Engineering"}},
			"user2": {{Id: "team1", DisplayName: "Engineering"}, {Id: "team2", DisplayName: "Sales"}},
			"user3": {{Id: "team1", DisplayName: "Engineering"}},
		},
		channels: map[string][]*model.Channel{
			"team1:user1": {{Id: "ch1", DisplayName: "General"}},
			"team1:user2": {{Id: "ch1", DisplayName: "General"}, {Id: "ch2", DisplayName: "Dev Backend"}},
			"team2:user2": {{Id: "ch3", DisplayName: "Partner Updates"}},
			"team1:user3": {{Id: "ch1", DisplayName: "General"}},
		},
		lastPostDate: map[string]*time.Time{
			"user1": timePtr(now.AddDate(0, 0, -2)),
			"user2": timePtr(now.AddDate(0, 0, -10)),
		},
	}

	result, exitCode := RunAudit(client, "", 0, false)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Guests) != 3 {
		t.Fatalf("expected 3 guests, got %d", len(result.Guests))
	}

	// Verify first guest
	g := result.Guests[0]
	if g.Username != "jane.doe" {
		t.Errorf("expected username jane.doe, got %s", g.Username)
	}
	if g.DisplayName != "Jane Doe" {
		t.Errorf("expected display name 'Jane Doe', got %q", g.DisplayName)
	}
	if len(g.Teams) != 1 {
		t.Errorf("expected 1 team, got %d", len(g.Teams))
	}
	if len(g.Channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(g.Channels))
	}
	if !g.Active {
		t.Error("expected guest to be active")
	}

	// Verify deactivated guest
	g3 := result.Guests[2]
	if g3.Active {
		t.Error("expected guest to be deactivated")
	}

	// Verify summary
	if result.Summary.TotalGuests != 3 {
		t.Errorf("expected total 3, got %d", result.Summary.TotalGuests)
	}
	if result.Summary.ActiveGuests != 2 {
		t.Errorf("expected 2 active, got %d", result.Summary.ActiveGuests)
	}
	if result.Summary.DeactivatedGuests != 1 {
		t.Errorf("expected 1 deactivated, got %d", result.Summary.DeactivatedGuests)
	}
}

func TestRunAudit_TeamFilter(t *testing.T) {
	now := time.Now()
	loginTime := now.AddDate(0, 0, -5)

	client := &mockClient{
		guests: []*model.User{
			{Id: "user1", Username: "jane.doe", FirstName: "Jane", LastName: "Doe", Email: "jane@example.com", CreateAt: 1709280000000, LastActivityAt: loginTime.UnixMilli()},
			{Id: "user2", Username: "bob.smith", FirstName: "Bob", LastName: "Smith", Email: "bob@example.com", CreateAt: 1709280000000, LastActivityAt: loginTime.UnixMilli()},
		},
		teamByName: map[string]*model.Team{
			"Sales": {Id: "team2", DisplayName: "Sales"},
		},
		teams: map[string][]*model.Team{
			"user1": {{Id: "team1", DisplayName: "Engineering"}},
			"user2": {{Id: "team1", DisplayName: "Engineering"}, {Id: "team2", DisplayName: "Sales"}},
		},
		channels: map[string][]*model.Channel{
			"team2:user2": {{Id: "ch3", DisplayName: "Partner Updates"}},
		},
		lastPostDate: map[string]*time.Time{
			"user2": timePtr(now.AddDate(0, 0, -3)),
		},
	}

	result, exitCode := RunAudit(client, "Sales", 0, false)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if len(result.Guests) != 1 {
		t.Fatalf("expected 1 guest in Sales team, got %d", len(result.Guests))
	}
	if result.Guests[0].Username != "bob.smith" {
		t.Errorf("expected bob.smith, got %s", result.Guests[0].Username)
	}
}

func TestRunAudit_TeamFilterNotFound(t *testing.T) {
	client := &mockClient{
		teamByName: map[string]*model.Team{},
	}

	result, exitCode := RunAudit(client, "NonExistent", 0, false)

	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitCode)
	}
	if result != nil {
		t.Error("expected nil result for unknown team")
	}
}

func TestRunAudit_Pagination(t *testing.T) {
	// Create 250 guests (page 0: 200, page 1: 50)
	guests := make([]*model.User, 250)
	teams := make(map[string][]*model.Team)
	channels := make(map[string][]*model.Channel)
	for i := 0; i < 250; i++ {
		id := fmt.Sprintf("user%d", i)
		guests[i] = &model.User{
			Id:       id,
			Username: fmt.Sprintf("guest%d", i),
			Email:    fmt.Sprintf("guest%d@example.com", i),
			CreateAt: 1709280000000,
		}
		teams[id] = []*model.Team{{Id: "team1", DisplayName: "Engineering"}}
		channels["team1:"+id] = []*model.Channel{{Id: "ch1", DisplayName: "General"}}
	}

	client := &mockClient{
		guests:   guests,
		teams:    teams,
		channels: channels,
	}

	result, exitCode := RunAudit(client, "", 0, false)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if len(result.Guests) != 250 {
		t.Errorf("expected 250 guests, got %d", len(result.Guests))
	}
}

func TestRunAudit_EmptyResult(t *testing.T) {
	client := &mockClient{
		guests: []*model.User{},
	}

	result, exitCode := RunAudit(client, "", 0, false)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if len(result.Guests) != 0 {
		t.Errorf("expected 0 guests, got %d", len(result.Guests))
	}
}

func TestRunAudit_PartialFailure(t *testing.T) {
	now := time.Now()
	loginTime := now.AddDate(0, 0, -5)

	client := &mockClient{
		guests: []*model.User{
			{Id: "user1", Username: "jane.doe", FirstName: "Jane", LastName: "Doe", Email: "jane@example.com", CreateAt: 1709280000000, LastActivityAt: loginTime.UnixMilli()},
			{Id: "user2", Username: "bob.smith", FirstName: "Bob", LastName: "Smith", Email: "bob@example.com", CreateAt: 1709280000000, LastActivityAt: loginTime.UnixMilli()},
		},
		teams: map[string][]*model.Team{
			"user1": {{Id: "team1", DisplayName: "Engineering"}},
		},
		teamsErr: map[string]error{
			"user2": fmt.Errorf("error: API request failed (HTTP 500)"),
		},
		channels: map[string][]*model.Channel{
			"team1:user1": {{Id: "ch1", DisplayName: "General"}},
		},
	}

	result, exitCode := RunAudit(client, "", 0, false)

	if exitCode != ExitPartialFailure {
		t.Errorf("expected exit code %d, got %d", ExitPartialFailure, exitCode)
	}
	if len(result.Guests) != 2 {
		t.Fatalf("expected 2 guests, got %d", len(result.Guests))
	}
	if result.Summary.FailedLookups != 1 {
		t.Errorf("expected 1 failed lookup, got %d", result.Summary.FailedLookups)
	}
	if result.Guests[1].Error == "" {
		t.Error("expected error message on failed guest")
	}
}

func TestRunAudit_InactivityFlagging(t *testing.T) {
	now := time.Now()

	client := &mockClient{
		guests: []*model.User{
			{Id: "user1", Username: "active.user", Email: "a@example.com", CreateAt: 1709280000000, LastActivityAt: now.AddDate(0, 0, -5).UnixMilli()},
			{Id: "user2", Username: "inactive.user", Email: "b@example.com", CreateAt: 1709280000000, LastActivityAt: now.AddDate(0, 0, -45).UnixMilli()},
			{Id: "user3", Username: "never.login", Email: "c@example.com", CreateAt: 1709280000000, LastActivityAt: 0},
		},
		teams: map[string][]*model.Team{
			"user1": {{Id: "team1", DisplayName: "Engineering"}},
			"user2": {{Id: "team1", DisplayName: "Engineering"}},
			"user3": {{Id: "team1", DisplayName: "Engineering"}},
		},
		channels: map[string][]*model.Channel{
			"team1:user1": {{Id: "ch1", DisplayName: "General"}},
			"team1:user2": {{Id: "ch1", DisplayName: "General"}},
			"team1:user3": {{Id: "ch1", DisplayName: "General"}},
		},
	}

	result, exitCode := RunAudit(client, "", 30, false)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	// Active user — not inactive
	if result.Guests[0].Inactive {
		t.Error("user1 should not be inactive")
	}
	// Inactive user — 45 days > 30 threshold
	if !result.Guests[1].Inactive {
		t.Error("user2 should be inactive")
	}
	// Never logged in — should be inactive
	if !result.Guests[2].Inactive {
		t.Error("user3 (never logged in) should be inactive")
	}

	if result.Summary.ActiveGuests != 1 {
		t.Errorf("expected 1 active guest, got %d", result.Summary.ActiveGuests)
	}
	if result.Summary.InactiveGuests != 2 {
		t.Errorf("expected 2 inactive guests, got %d", result.Summary.InactiveGuests)
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
