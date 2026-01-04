package github

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// mockCommandRunner is a mock implementation of the commandRunner interface for testing.
type mockCommandRunner struct {
	runFunc func(name string, args ...string) ([]byte, error)
}

func (m *mockCommandRunner) run(name string, args ...string) ([]byte, error) {
	if m.runFunc != nil {
		return m.runFunc(name, args...)
	}
	return nil, fmt.Errorf("runFunc not set for mockCommandRunner")
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name         string
		mockOutput   []byte
		mockError    error
		expectedUser string
		expectedErr  string
	}{
		{
			name:         "Success",
			mockOutput:   []byte("github.com\n  ✓ Logged in to github.com account testuser (keyring)"),
			mockError:    nil,
			expectedUser: "testuser",
			expectedErr:  "",
		},
		{
			name:         "gh command error",
			mockOutput:   nil,
			mockError:    errors.New("command failed"),
			expectedUser: "",
			expectedErr:  "failed to run 'gh auth status'",
		},
		{
			name:         "Unexpected output format",
			mockOutput:   []byte("some unexpected output"),
			mockError:    nil,
			expectedUser: "",
			expectedErr:  "could not find authenticated user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockCommandRunner{
				runFunc: func(name string, args ...string) ([]byte, error) {
					return tt.mockOutput, tt.mockError
				},
			}
			client := NewClientWithRunner(runner)

			user, err := client.GetUser()

			if tt.expectedErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if user != tt.expectedUser {
					t.Errorf("expected user '%s', got '%s'", tt.expectedUser, user)
				}
			}
		})
	}
}

func TestGetFollowing(t *testing.T) {
	tests := []struct {
		name        string
		user        string
		mockOutput  []byte
		mockError   error
		expected    []string
		expectedErr string
	}{
		{
			name:       "Success",
			user:       "testuser",
			mockOutput: []byte(`[{"login": "alice"}, {"login": "bob"}]`),
			mockError:  nil,
			expected:   []string{"alice", "bob"},
		},
		{
			name:        "gh command error",
			user:        "testuser",
			mockError:   errors.New("command failed"),
			expectedErr: "failed to run 'gh api users/testuser/following'",
		},
		{
			name:        "Invalid JSON",
			user:        "testuser",
			mockOutput:  []byte(`[{"login": "alice", "invalid": "json"`),
			expectedErr: "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockCommandRunner{
				runFunc: func(name string, args ...string) ([]byte, error) {
					return tt.mockOutput, tt.mockError
				},
			}
			client := NewClientWithRunner(runner)

			following, err := client.GetFollowing(tt.user)

			if tt.expectedErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !compareStringSlices(following, tt.expected) {
					t.Errorf("expected %v, got %v", tt.expected, following)
				}
			}
		})
	}
}

func TestGetFollowers(t *testing.T) {
    // Similar structure to TestGetFollowing
	tests := []struct {
		name        string
		user        string
		mockOutput  []byte
		mockError   error
		expected    []string
		expectedErr string
	}{
		{
			name:       "Success",
			user:       "testuser",
			mockOutput: []byte(`[{"login": "charlie"}, {"login": "dave"}]`),
			expected:   []string{"charlie", "dave"},
		},
		{
			name:        "gh command error",
			user:        "testuser",
			mockError:   errors.New("command failed"),
			expectedErr: "failed to run 'gh api users/testuser/followers'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockCommandRunner{
				runFunc: func(name string, args ...string) ([]byte, error) {
					return tt.mockOutput, tt.mockError
				},
			}
			client := NewClientWithRunner(runner)

			followers, err := client.GetFollowers(tt.user)

			if tt.expectedErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !compareStringSlices(followers, tt.expected) {
					t.Errorf("expected %v, got %v", tt.expected, followers)
				}
			}
		})
	}
}

func TestUnfollow(t *testing.T) {
	tests := []struct {
		name        string
		user        string
		mockError   error
		expectedErr string
	}{
		{
			name: "Success",
			user: "testuser",
		},
		{
			name:        "gh command error",
			user:        "testuser",
			mockError:   errors.New("command failed"),
			expectedErr: "failed to unfollow testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockCommandRunner{
				runFunc: func(name string, args ...string) ([]byte, error) {
					return nil, tt.mockError
				},
			}
			client := NewClientWithRunner(runner)

			err := client.Unfollow(tt.user)

			if tt.expectedErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedErr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFollow(t *testing.T) {
	tests := []struct {
		name        string
		user        string
		mockError   error
		expectedErr string
	}{
		{
			name: "Success",
			user: "testuser",
		},
		{
			name:        "gh command error",
			user:        "testuser",
			mockError:   errors.New("command failed"),
			expectedErr: "failed to follow testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockCommandRunner{
				runFunc: func(name string, args ...string) ([]byte, error) {
					return nil, tt.mockError
				},
			}
			client := NewClientWithRunner(runner)

			err := client.Follow(tt.user)

			if tt.expectedErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedErr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Helper to compare two string slices
func compareStringSlices(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	sort.Strings(s1)
	sort.Strings(s2)
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func TestGetMutualFollowsData(t *testing.T) {
	tests := []struct {
		name                  string
		following             []string
		followers             []string
		expectedOnlyFollowing []string
		expectedOnlyFollowers []string
	}{
		{
			name:                  "Standard case",
			following:             []string{"a", "b", "c"},
			followers:             []string{"b", "c", "d"},
			expectedOnlyFollowing: []string{"a"},
			expectedOnlyFollowers: []string{"d"},
		},
		{
			name:                  "Empty following",
			following:             []string{},
			followers:             []string{"a", "b"},
			expectedOnlyFollowing: []string{},
			expectedOnlyFollowers: []string{"a", "b"},
		},
		{
			name:                  "Empty followers",
			following:             []string{"a", "b"},
			followers:             []string{},
			expectedOnlyFollowing: []string{"a", "b"},
			expectedOnlyFollowers: []string{},
		},
		{
			name:                  "All mutual",
			following:             []string{"a", "b"},
			followers:             []string{"a", "b"},
			expectedOnlyFollowing: []string{},
			expectedOnlyFollowers: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onlyFollowing, onlyFollowers := GetMutualFollowsData("user", tt.following, tt.followers)

			if !compareStringSlices(onlyFollowing, tt.expectedOnlyFollowing) {
				t.Errorf("Expected onlyFollowing %v, got %v", tt.expectedOnlyFollowing, onlyFollowing)
			}
			if !compareStringSlices(onlyFollowers, tt.expectedOnlyFollowers) {
				t.Errorf("Expected onlyFollowers %v, got %v", tt.expectedOnlyFollowers, onlyFollowers)
			}
		})
	}
}
