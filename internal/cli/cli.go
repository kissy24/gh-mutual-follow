package cli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
)

// runCommand is a helper function to execute shell commands.
// It can be mocked in tests.
var runCommand = func(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// GitHubUser represents a simplified GitHub user for JSON unmarshalling.
type GitHubUser struct {
	Login string `json:"login"`
}

// GetUser returns the GitHub username of the authenticated user.
func GetUser() (string, error) {
	output, err := runCommand("gh", "auth", "status")
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
func GetFollowing(user string) ([]string, error) {
	output, err := runCommand("gh", "api", "--paginate", "users/"+user+"/following")
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
func GetFollowers(user string) ([]string, error) {
	output, err := runCommand("gh", "api", "--paginate", "users/"+user+"/followers")
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

// GetMutualFollowsData calculates the 'only following' and 'only followers' lists.
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


// Unfollow unfollows a given user.
func Unfollow(user string) error {
	_, err := runCommand("gh", "api", "--method", "DELETE", "user/following/"+user)
	if err != nil {
		return fmt.Errorf("failed to unfollow %s: %w", user, err)
	}
	return nil
}

// Follow follows a given user.
func Follow(user string) error {
	_, err := runCommand("gh", "api", "--method", "PUT", "user/following/"+user)
	if err != nil {
		return fmt.Errorf("failed to follow %s: %w", user, err)
	}
	return nil
}