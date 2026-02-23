package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"golang.org/x/term"
)

// MattermostClient abstracts the Mattermost API calls needed by mm-guest-audit.
// This interface enables unit testing with mock implementations.
type MattermostClient interface {
	GetGuestUsers(page, perPage int) ([]*model.User, error)
	GetTeamByName(name string) (*model.Team, error)
	GetTeamsForUser(userID string) ([]*model.Team, error)
	GetChannelsForTeamForUser(teamID, userID string) ([]*model.Channel, error)
	GetLastPostDateForUser(userID, username string, teamIDs []string) (*time.Time, error)
}

// mmClient is the real implementation backed by model.Client4.
type mmClient struct {
	api *model.Client4
	ctx context.Context
}

// NormalizeURL strips trailing slashes from the server URL.
func NormalizeURL(url string) string {
	return strings.TrimRight(url, "/")
}

// NewClient creates a new Mattermost API client and authenticates.
func NewClient(url, token, username string, verbose bool) (MattermostClient, error) {
	url = NormalizeURL(url)
	api := model.NewAPIv4Client(url)
	ctx := context.Background()

	if token != "" {
		api.SetToken(token)
		if verbose {
			fmt.Fprintln(os.Stderr, "Authenticating with personal access token...")
		}
		// Verify the token works
		_, resp, err := api.GetMe(ctx, "")
		if err != nil {
			return nil, classifyAPIError(url, resp, err)
		}
	} else if username != "" {
		password, err := obtainPassword()
		if err != nil {
			return nil, err
		}
		if verbose {
			fmt.Fprintln(os.Stderr, "Authenticating with username and password...")
		}
		_, resp, err := api.Login(ctx, username, password)
		if err != nil {
			return nil, classifyAPIError(url, resp, err)
		}
	} else {
		return nil, fmt.Errorf("error: authentication required. Use --token (or MM_TOKEN) for token auth, or --username (or MM_USERNAME) for password auth")
	}

	return &mmClient{api: api, ctx: ctx}, nil
}

// obtainPassword gets the password from TTY prompt or MM_PASSWORD env var.
func obtainPassword() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // move to next line after input
		if err != nil {
			return "", fmt.Errorf("error: failed to read password: %w", err)
		}
		return string(passwordBytes), nil
	}

	password := os.Getenv("MM_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("error: password required. Set MM_PASSWORD for non-interactive use, or run interactively to be prompted")
	}
	return password, nil
}

func (c *mmClient) GetGuestUsers(page, perPage int) ([]*model.User, error) {
	users, resp, err := c.api.GetUsersWithCustomQueryParameters(c.ctx, page, perPage, "role=system_guest", "")
	if err != nil {
		return nil, classifyAPIError("", resp, err)
	}
	return users, nil
}

func (c *mmClient) GetTeamByName(name string) (*model.Team, error) {
	team, resp, err := c.api.GetTeamByName(c.ctx, name, "")
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, fmt.Errorf("error: team %q not found. Please check the name and try again", name)
		}
		return nil, classifyAPIError("", resp, err)
	}
	return team, nil
}

func (c *mmClient) GetTeamsForUser(userID string) ([]*model.Team, error) {
	teams, resp, err := c.api.GetTeamsForUser(c.ctx, userID, "")
	if err != nil {
		return nil, classifyAPIError("", resp, err)
	}
	return teams, nil
}

func (c *mmClient) GetChannelsForTeamForUser(teamID, userID string) ([]*model.Channel, error) {
	channels, resp, err := c.api.GetChannelsForTeamForUser(c.ctx, teamID, userID, false, "")
	if err != nil {
		return nil, classifyAPIError("", resp, err)
	}
	return channels, nil
}

func (c *mmClient) GetLastPostDateForUser(userID, username string, teamIDs []string) (*time.Time, error) {
	var latestTime *time.Time

	for _, teamID := range teamIDs {
		posts, resp, err := c.api.SearchPosts(c.ctx, teamID, "from:"+username, false)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				continue
			}
			return nil, classifyAPIError("", resp, err)
		}
		if posts == nil {
			continue
		}
		for _, post := range posts.Posts {
			t := MillisToTime(post.CreateAt)
			if t != nil && (latestTime == nil || t.After(*latestTime)) {
				latestTime = t
			}
		}
	}

	return latestTime, nil
}

// ClassifyAPIError maps API response status codes to human-readable error messages.
func ClassifyAPIError(url string, statusCode int) error {
	return classifyAPIErrorFromStatus(url, statusCode)
}

func classifyAPIError(url string, resp *model.Response, err error) error {
	if resp == nil {
		if url != "" {
			return fmt.Errorf("error: unable to connect to %s. Check the URL and network connectivity", url)
		}
		return fmt.Errorf("error: API request failed: %w", err)
	}
	return classifyAPIErrorFromStatus(url, resp.StatusCode)
}

func classifyAPIErrorFromStatus(url string, statusCode int) error {
	switch {
	case statusCode == 401:
		return fmt.Errorf("error: authentication failed. Check your token or credentials")
	case statusCode == 403:
		return fmt.Errorf("error: permission denied. This operation requires a System Administrator account")
	case statusCode == 404:
		return fmt.Errorf("error: the requested resource was not found")
	case statusCode >= 500:
		return fmt.Errorf("error: the Mattermost server returned an unexpected error (HTTP %d). Check server logs for details", statusCode)
	default:
		return fmt.Errorf("error: API request failed (HTTP %d)", statusCode)
	}
}
