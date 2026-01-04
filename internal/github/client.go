package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Client defines the interface for interacting with the GitHub API.
type Client interface {
	GetUser() (string, error)
	GetFollowing(user string) ([]string, error)
	GetFollowers(user string) ([]string, error)
	Unfollow(user string) error
	Follow(user string) error
}

// commandRunner defines an interface for running external commands.
// This makes the client testable by allowing mock runners.
type commandRunner interface {
	run(name string, args ...string) ([]byte, error)
}

// execCommandRunner is the concrete implementation of commandRunner that uses os/exec.
type execCommandRunner struct{}

func (r *execCommandRunner) run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("command '%s %s' failed with exit code %d: %s (stderr: %s)",
				name, strings.Join(args, " "), exitErr.ExitCode(), err, stderr.String())
		}
		return nil, fmt.Errorf("command '%s %s' failed: %w (stderr: %s)",
			name, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// ghClient is the concrete implementation of the Client interface.
type ghClient struct {
	runner commandRunner
}

// NewClient creates a new instance of ghClient with the default command runner.
func NewClient() Client {
	return &ghClient{runner: &execCommandRunner{}}
}

// NewClientWithRunner is a constructor for testing, allowing a mock runner to be injected.
func NewClientWithRunner(runner commandRunner) Client {
	return &ghClient{runner: runner}
}


// GitHubUser represents a simplified GitHub user for JSON unmarshalling.
type GitHubUser struct {
	Login string `json:"login"`
}

// GetUser returns the GitHub username of the authenticated user.
func (c *ghClient) GetUser() (string, error) {
	output, err := c.runner.run("gh", "auth", "status")
	if err != nil {
		return "", fmt.Errorf("failed to run 'gh auth status': %w", err)
	}

	re := regexp.MustCompile(`Logged in to github.com account (\S+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find authenticated user in 'gh auth status' output")
	}

	return matches[1], nil
}

// GetFollowing returns a list of users that the given user is following.
func (c *ghClient) GetFollowing(user string) ([]string, error) {
	output, err := c.runner.run("gh", "api", "--paginate", "users/"+user+"/following")
	if err != nil {
		return nil, fmt.Errorf("failed to run 'gh api users/%s/following': %w", user, err)
	}

	var users []GitHubUser
	if err := json.Unmarshal(output, &users); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from 'gh api users/%s/following': %w", user, err)
	}

	var following []string
	for _, u := range users {
		following = append(following, u.Login)
	}
	return following, nil
}

// GetFollowers returns a list of users that are following the given user.
func (c *ghClient) GetFollowers(user string) ([]string, error) {
	output, err := c.runner.run("gh", "api", "--paginate", "users/"+user+"/followers")
	if err != nil {
		return nil, fmt.Errorf("failed to run 'gh api users/%s/followers': %w", user, err)
	}

	var users []GitHubUser
	if err := json.Unmarshal(output, &users); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from 'gh api users/%s/followers': %w", user, err)
	}

	var followers []string
	for _, u := range users {
		followers = append(followers, u.Login)
	}
	return followers, nil
}

// Unfollow unfollows a given user.
func (c *ghClient) Unfollow(user string) error {
	_, err := c.runner.run("gh", "api", "--method", "DELETE", "user/following/"+user)
	if err != nil {
		return fmt.Errorf("failed to unfollow %s: %w", user, err)
	}
	return nil
}

// Follow follows a given user.
func (c *ghClient) Follow(user string) error {
	_, err := c.runner.run("gh", "api", "--method", "PUT", "user/following/"+user)
	if err != nil {
		return fmt.Errorf("failed to follow %s: %w", user, err)
	}
	return nil
}

// GetMutualFollowsData calculates the 'only following' and 'only followers' lists.
// This is a pure function and does not need to be a method on the client.
func GetMutualFollowsData(authenticatedUser string, following, followers []string) (onlyFollowing []string, onlyFollowers []string) {
	followingMap := make(map[string]bool)
	for _, u := range following {
		followingMap[u] = true
	}

	followersMap := make(map[string]bool)
	for _, u := range followers {
		followersMap[u] = true
	}

	// Use temporary maps to collect unique results
	uniqueOnlyFollowing := make(map[string]bool)
	for _, u := range following {
		if _, exists := followersMap[u]; !exists {
			uniqueOnlyFollowing[u] = true
		}
	}

	uniqueOnlyFollowers := make(map[string]bool)
	for _, u := range followers {
		if _, exists := followingMap[u]; !exists {
			uniqueOnlyFollowers[u] = true
		}
	}

	// Convert maps back to slices
	for u := range uniqueOnlyFollowing {
		onlyFollowing = append(onlyFollowing, u)
	}
	for u := range uniqueOnlyFollowers {
		onlyFollowers = append(onlyFollowers, u)
	}

	return onlyFollowing, onlyFollowers
}