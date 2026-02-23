package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sampleResult() *AuditResult {
	created := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)
	login := time.Date(2024, 11, 15, 8, 32, 0, 0, time.UTC)
	post := time.Date(2024, 11, 14, 17, 22, 0, 0, time.UTC)

	return &AuditResult{
		InactiveDays: 30,
		Guests: []GuestRecord{
			{
				Username:    "jane.doe",
				DisplayName: "Jane Doe",
				Email:       "jane.doe@external.com",
				CreatedAt:   &created,
				LastLogin:   &login,
				LastPost:    &post,
				Teams: []TeamInfo{
					{ID: "team1", DisplayName: "Engineering"},
					{ID: "team2", DisplayName: "Sales"},
				},
				Channels: []ChannelInfo{
					{TeamName: "Engineering", ChannelName: "General"},
					{TeamName: "Engineering", ChannelName: "Dev Backend"},
					{TeamName: "Sales", ChannelName: "Partner Updates"},
				},
				Active:   true,
				Inactive: false,
			},
			{
				Username:    "bob.contractor",
				DisplayName: "Bob Contractor",
				Email:       "bob@contractor.io",
				CreatedAt:   &created,
				LastLogin:   nil,
				LastPost:    nil,
				Teams: []TeamInfo{
					{ID: "team1", DisplayName: "Engineering"},
				},
				Channels: []ChannelInfo{
					{TeamName: "Engineering", ChannelName: "General"},
				},
				Active:   true,
				Inactive: true,
			},
		},
		Summary: AuditSummary{
			TotalGuests:    2,
			ActiveGuests:   1,
			InactiveGuests: 1,
		},
	}
}

func TestFormatCSV(t *testing.T) {
	result := sampleResult()
	var buf bytes.Buffer
	err := writeCSV(&buf, result)
	if err != nil {
		t.Fatalf("writeCSV error: %v", err)
	}

	output := buf.String()

	// Parse CSV to verify structure
	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}

	// Header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 rows (header + 2 data), got %d", len(records))
	}

	// Verify header
	expectedHeader := []string{"username", "display_name", "email", "created_at", "last_login", "last_post", "teams", "channels", "active", "inactive"}
	for i, h := range expectedHeader {
		if records[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, records[0][i], h)
		}
	}

	// Verify first data row
	row := records[1]
	if row[0] != "jane.doe" {
		t.Errorf("username = %q, want 'jane.doe'", row[0])
	}
	if row[3] != "2024-03-01T10:00:00Z" {
		t.Errorf("created_at = %q, want ISO 8601 date", row[3])
	}
	// Teams should be pipe-separated
	if row[6] != "Engineering|Sales" {
		t.Errorf("teams = %q, want 'Engineering|Sales'", row[6])
	}
	// Channels should be team/channel pipe-separated
	if row[7] != "Engineering/General|Engineering/Dev Backend|Sales/Partner Updates" {
		t.Errorf("channels = %q, want pipe-separated team/channel pairs", row[7])
	}
	if row[8] != "true" {
		t.Errorf("active = %q, want 'true'", row[8])
	}
	if row[9] != "false" {
		t.Errorf("inactive = %q, want 'false'", row[9])
	}

	// Verify second data row (nil dates)
	row2 := records[2]
	if row2[4] != "" {
		t.Errorf("last_login for nil date = %q, want empty string", row2[4])
	}
	if row2[5] != "" {
		t.Errorf("last_post for nil date = %q, want empty string", row2[5])
	}
	if row2[9] != "true" {
		t.Errorf("inactive = %q, want 'true'", row2[9])
	}
}

func TestFormatCSV_EmptyResult(t *testing.T) {
	result := &AuditResult{
		Guests:  []GuestRecord{},
		Summary: AuditSummary{},
	}

	var buf bytes.Buffer
	err := writeCSV(&buf, result)
	if err != nil {
		t.Fatalf("writeCSV error: %v", err)
	}

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}

	// Header only, no data
	if len(records) != 1 {
		t.Errorf("expected 1 row (header only), got %d", len(records))
	}
}

func TestFormatJSON(t *testing.T) {
	result := sampleResult()
	var buf bytes.Buffer
	err := writeJSON(&buf, result)
	if err != nil {
		t.Fatalf("writeJSON error: %v", err)
	}

	// Verify it's valid JSON
	var output jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("JSON parse error: %v\nRaw output:\n%s", err, buf.String())
	}

	// Summary
	if output.Summary.TotalGuests != 2 {
		t.Errorf("summary.total_guests = %d, want 2", output.Summary.TotalGuests)
	}
	if output.Summary.ActiveGuests != 1 {
		t.Errorf("summary.active_guests = %d, want 1", output.Summary.ActiveGuests)
	}
	if output.InactiveDays != 30 {
		t.Errorf("inactive_days = %d, want 30", output.InactiveDays)
	}

	// Guest fields
	if len(output.Guests) != 2 {
		t.Fatalf("expected 2 guests, got %d", len(output.Guests))
	}

	g := output.Guests[0]
	if g.Username != "jane.doe" {
		t.Errorf("username = %q, want 'jane.doe'", g.Username)
	}
	if len(g.Teams) != 2 {
		t.Errorf("expected 2 teams, got %d", len(g.Teams))
	}
	if len(g.Channels) != 3 {
		t.Errorf("expected 3 channels, got %d", len(g.Channels))
	}
	if g.Channels[0].TeamName != "Engineering" || g.Channels[0].ChannelName != "General" {
		t.Errorf("first channel = %+v, want Engineering/General", g.Channels[0])
	}
}

func TestFormatJSON_NilDates(t *testing.T) {
	result := sampleResult()
	var buf bytes.Buffer
	err := writeJSON(&buf, result)
	if err != nil {
		t.Fatalf("writeJSON error: %v", err)
	}

	// Verify null dates in raw JSON
	raw := buf.String()
	var output jsonOutput
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	// Second guest has nil dates
	g := output.Guests[1]
	if g.LastLogin != nil {
		t.Errorf("last_login should be null, got %v", *g.LastLogin)
	}
	if g.LastPost != nil {
		t.Errorf("last_post should be null, got %v", *g.LastPost)
	}

	// Verify it contains "null" in the raw JSON for these fields
	if !strings.Contains(raw, `"last_login": null`) {
		t.Error("expected JSON null for last_login, not found in raw output")
	}
}

func TestFormatTable(t *testing.T) {
	result := sampleResult()
	var buf bytes.Buffer
	err := writeTable(&buf, result)
	if err != nil {
		t.Fatalf("writeTable error: %v", err)
	}

	output := buf.String()

	// Verify header row
	if !strings.Contains(output, "USERNAME") {
		t.Error("table missing USERNAME header")
	}
	if !strings.Contains(output, "DISPLAY NAME") {
		t.Error("table missing DISPLAY NAME header")
	}

	// Verify data
	if !strings.Contains(output, "jane.doe") {
		t.Error("table missing jane.doe")
	}
	if !strings.Contains(output, "Jane Doe") {
		t.Error("table missing display name")
	}

	// Verify channel truncation (3 channels, max display 2)
	if !strings.Contains(output, "(+1 more)") {
		t.Error("table should truncate channels with (+N more)")
	}

	// Verify summary
	if !strings.Contains(output, "Total: 2 guest(s)") {
		t.Error("table missing summary line")
	}

	// Verify status
	if !strings.Contains(output, "Active") {
		t.Error("table missing Active status")
	}
	if !strings.Contains(output, "Inactive") {
		t.Error("table missing Inactive status")
	}
}

func TestFormatTable_ChannelTruncation(t *testing.T) {
	result := &AuditResult{
		Guests: []GuestRecord{
			{
				Username: "test.user",
				Channels: []ChannelInfo{
					{TeamName: "Team", ChannelName: "Alpha"},
					{TeamName: "Team", ChannelName: "Beta"},
					{TeamName: "Team", ChannelName: "Gamma"},
					{TeamName: "Team", ChannelName: "Delta"},
					{TeamName: "Team", ChannelName: "Epsilon"},
				},
				Active: true,
			},
		},
		Summary: AuditSummary{TotalGuests: 1, ActiveGuests: 1},
	}

	var buf bytes.Buffer
	writeTable(&buf, result)
	output := buf.String()

	if !strings.Contains(output, "(+3 more)") {
		t.Errorf("expected (+3 more) truncation, got:\n%s", output)
	}
}
