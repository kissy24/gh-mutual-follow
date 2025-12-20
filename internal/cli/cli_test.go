package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// withMockedCommand temporarily replaces the global runCommand with a mock version.
func withMockedCommand(mock func(name string, args ...string) ([]byte, error), testFunc func()) {
	oldRunCommand := runCommand
	runCommand = mock
	defer func() {
		runCommand = oldRunCommand
	}()
	testFunc()
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name         string
		mockOutput   []byte
		mockError    error
		expectedUser string
		expectedErr  error
	}{
		{
			name:         "Success",
			mockOutput:   []byte("github.com\n  ✓ Logged in to github.com account testuser (keyring)"),
			mockError:    nil,
			expectedUser: "testuser",
			expectedErr:  nil,
		},
		{
			name:         "gh command error",
			mockOutput:   nil,
			mockError:    errors.New("command failed"),
			expectedUser: "",
			expectedErr:  fmt.Errorf("failed to run 'gh auth status': command failed"),
		},
		{
			name:         "Unexpected output format",
			mockOutput:   []byte("some unexpected output"),
			mockError:    nil,
			expectedUser: "",
			expectedErr:  fmt.Errorf("could not find authenticated user in 'gh auth status' output"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMockedCommand(func(name string, args ...string) ([]byte, error) {
				if name == "gh" && args[0] == "auth" && args[1] == "status" {
					return tt.mockOutput, tt.mockError
				}
				return nil, fmt.Errorf("unexpected command: %s %v", name, args)
			}, func() {
				user, err := GetUser()

				if tt.expectedErr != nil {
					if err == nil || !strings.Contains(err.Error(), tt.expectedErr.Error()) {
						t.Errorf("expected error %v, got %v", tt.expectedErr, err)
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
					if user != tt.expectedUser {
						t.Errorf("expected user %s, got %s", tt.expectedUser, user)
					}
				}
			})
		})
	}
}

func TestGetFollowing(t *testing.T) {
	tests := []struct {
		name            string
		user            string
		mockOutput      []byte
		mockError       error
		expected        []string
		expectedErr     error
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
			mockOutput:  nil,
			mockError:   errors.New("command failed"),
			expected:    nil,
			expectedErr: fmt.Errorf("failed to run 'gh api users/testuser/following': command failed"),
		},
		{
			name:        "Invalid JSON",
			user:        "testuser",
			mockOutput:  []byte(`[{"login": "alice", "invalid": "json"`),
			mockError:   nil,
			expected:    nil,
			expectedErr: fmt.Errorf("failed to parse JSON from 'gh api users/testuser/following'"),
		},
		{
			name:       "Empty list",
			user:       "testuser",
			mockOutput: []byte(`[]`),
			mockError:  nil,
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMockedCommand(func(name string, args ...string) ([]byte, error) {
				if name == "gh" && args[0] == "api" && args[1] == "users/"+tt.user+"/following" {
					return tt.mockOutput, tt.mockError
				}
				return nil, fmt.Errorf("unexpected command: %s %v", name, args)
			}, func() {
				following, err := GetFollowing(tt.user)

				if tt.expectedErr != nil {
					if err == nil || !strings.Contains(err.Error(), tt.expectedErr.Error()) {
						t.Errorf("expected error %v, got %v", tt.expectedErr, err)
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
		})
	}
}

func TestGetFollowers(t *testing.T) {
	tests := []struct {
		name            string
		user            string
		mockOutput      []byte
		mockError       error
		expected        []string
		expectedErr     error
	}{
		{
			name:       "Success",
			user:       "testuser",
			mockOutput: []byte(`[{"login": "charlie"}, {"login": "dave"}]`),
			mockError:  nil,
			expected:   []string{"charlie", "dave"},
		},
		{
			name:        "gh command error",
			user:        "testuser",
			mockOutput:  nil,
			mockError:   errors.New("command failed"),
			expected:    nil,
			expectedErr: fmt.Errorf("failed to run 'gh api users/testuser/followers': command failed"),
		},
		{
			name:        "Invalid JSON",
			user:        "testuser",
			mockOutput:  []byte(`[{"login": "charlie", "invalid": "json"`),
			mockError:   nil,
			expected:    nil,
			expectedErr: fmt.Errorf("failed to parse JSON from 'gh api users/testuser/followers'"),
		},
		{
			name:       "Empty list",
			user:       "testuser",
			mockOutput: []byte(`[]`),
			mockError:  nil,
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMockedCommand(func(name string, args ...string) ([]byte, error) {
				if name == "gh" && args[0] == "api" && args[1] == "users/"+tt.user+"/followers" {
					return tt.mockOutput, tt.mockError
				}
				return nil, fmt.Errorf("unexpected command: %s %v", name, args)
			}, func() {
				followers, err := GetFollowers(tt.user)

				if tt.expectedErr != nil {
					if err == nil || !strings.Contains(err.Error(), tt.expectedErr.Error()) {
						t.Errorf("expected error %v, got %v", tt.expectedErr, err)
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
		})
	}
}

func TestUnfollow(t *testing.T) {
	tests := []struct {
		name        string
		user        string
		mockOutput  []byte
		mockError   error
		expectedErr error
	}{
		{
			name:        "Success",
			user:        "testuser",
			mockOutput:  nil,
			mockError:   nil,
			expectedErr: nil,
		},
		{
			name:        "gh command error",
			user:        "testuser",
			mockOutput:  nil,
			mockError:   errors.New("command failed"),
			expectedErr: fmt.Errorf("failed to unfollow testuser: command failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMockedCommand(func(name string, args ...string) ([]byte, error) {
				if name == "gh" && args[0] == "api" && args[1] == "--method" && args[2] == "DELETE" && args[3] == "user/following/"+tt.user {
					return tt.mockOutput, tt.mockError
				}
				return nil, fmt.Errorf("unexpected command: %s %v", name, args)
			}, func() {
				err := Unfollow(tt.user)

				if tt.expectedErr != nil {
					if err == nil || !strings.Contains(err.Error(), tt.expectedErr.Error()) {
						t.Errorf("expected error %v, got %v", tt.expectedErr, err)
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}
			})
		})
	}
}

func TestFollow(t *testing.T) {
	tests := []struct {
		name        string
		user        string
		mockOutput  []byte
		mockError   error
		expectedErr error
	}{
		{
			name:        "Success",
			user:        "testuser",
			mockOutput:  nil,
			mockError:   nil,
			expectedErr: nil,
		},
		{
			name:        "gh command error",
			user:        "testuser",
			mockOutput:  nil,
			mockError:   errors.New("command failed"),
			expectedErr: fmt.Errorf("failed to follow testuser: command failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMockedCommand(func(name string, args ...string) ([]byte, error) {
				if name == "gh" && args[0] == "api" && args[1] == "--method" && args[2] == "PUT" && args[3] == "user/following/"+tt.user {
					return tt.mockOutput, tt.mockError
				}
				return nil, fmt.Errorf("unexpected command: %s %v", name, args)
			}, func() {
				err := Follow(tt.user)

				if tt.expectedErr != nil {
					if err == nil || !strings.Contains(err.Error(), tt.expectedErr.Error()) {
						t.Errorf("expected error %v, got %v", tt.expectedErr, err)
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}
			})
		})
	}
}

// Helper to compare two string slices
func compareStringSlices(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	// Sort slices to handle out-of-order elements
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
		name              string
		authenticatedUser string
		following         []string
		followers         []string
		expectedOnlyFollowing []string
		expectedOnlyFollowers []string
	}{
		{
			name:              "Standard case with mutual and non-mutual",
			authenticatedUser: "testuser",
			following:         []string{"alice", "bob", "charlie", "dave"},
			followers:         []string{"bob", "charlie", "eve", "frank"},
			expectedOnlyFollowing: []string{"alice", "dave"},
			expectedOnlyFollowers: []string{"eve", "frank"},
		},
		{
			name:              "No mutual follows",
			authenticatedUser: "testuser",
			following:         []string{"alice", "bob"},
			followers:         []string{"charlie", "dave"},
			expectedOnlyFollowing: []string{"alice", "bob"},
			expectedOnlyFollowers: []string{"charlie", "dave"},
		},
		{
			name:              "All mutual follows",
			authenticatedUser: "testuser",
			following:         []string{"alice", "bob"},
			followers:         []string{"alice", "bob"},
			expectedOnlyFollowing: []string{},
			expectedOnlyFollowers: []string{},
		},
		{
			name:              "Empty following list",
			authenticatedUser: "testuser",
			following:         []string{},
			followers:         []string{"alice", "bob"},
			expectedOnlyFollowing: []string{},
			expectedOnlyFollowers: []string{"alice", "bob"},
		},
		{
			name:              "Empty followers list",
			authenticatedUser: "testuser",
			following:         []string{"alice", "bob"},
			followers:         []string{},
			expectedOnlyFollowing: []string{"alice", "bob"},
			expectedOnlyFollowers: []string{},
		},
		{
			name:              "Both lists empty",
			authenticatedUser: "testuser",
			following:         []string{},
			followers:         []string{},
			expectedOnlyFollowing: []string{},
			expectedOnlyFollowers: []string{},
		},
		{
			name:              "Following list contains duplicates",
			authenticatedUser: "testuser",
			following:         []string{"alice", "bob", "alice"}, // Duplicates should be handled implicitly by map conversion
			followers:         []string{"bob", "charlie"},
			expectedOnlyFollowing: []string{"alice"},
			expectedOnlyFollowers: []string{"charlie"},
		},
		{
			name:              "Followers list contains duplicates",
			authenticatedUser: "testuser",
			following:         []string{"alice", "bob"},
			followers:         []string{"bob", "charlie", "charlie"}, // Duplicates should be handled implicitly by map conversion
			expectedOnlyFollowing: []string{"alice"},
			expectedOnlyFollowers: []string{"charlie"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onlyFollowing, onlyFollowers := GetMutualFollowsData(tt.authenticatedUser, tt.following, tt.followers)

			if !compareStringSlices(onlyFollowing, tt.expectedOnlyFollowing) {
				t.Errorf("Test %s: Expected onlyFollowing %v, got %v", tt.name, tt.expectedOnlyFollowing, onlyFollowing)
			}
			if !compareStringSlices(onlyFollowers, tt.expectedOnlyFollowers) {
				t.Errorf("Test %s: Expected onlyFollowers %v, got %v", tt.name, tt.expectedOnlyFollowers, onlyFollowers)
			}
		})
	}
}
