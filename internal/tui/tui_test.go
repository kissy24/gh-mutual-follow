package tui

import (
	"errors"
	"testing"
	// "gh-mutual-follow/internal/github" // Kept for future tests

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockGitHubClient is a mock implementation of the github.Client interface for testing.
type mockGitHubClient struct {
	GetUserFunc      func() (string, error)
	GetFollowingFunc func(user string) ([]string, error)
	GetFollowersFunc func(user string) ([]string, error)
	UnfollowFunc     func(user string) error
	FollowFunc       func(user string) error
}

func (m *mockGitHubClient) GetUser() (string, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc()
	}
	return "", errors.New("GetUserFunc not implemented")
}

func (m *mockGitHubClient) GetFollowing(user string) ([]string, error) {
	if m.GetFollowingFunc != nil {
		return m.GetFollowingFunc(user)
	}
	return nil, errors.New("GetFollowingFunc not implemented")
}

func (m *mockGitHubClient) GetFollowers(user string) ([]string, error) {
	if m.GetFollowersFunc != nil {
		return m.GetFollowersFunc(user)
	}
	return nil, errors.New("GetFollowersFunc not implemented")
}

func (m *mockGitHubClient) Unfollow(user string) error {
	if m.UnfollowFunc != nil {
		return m.UnfollowFunc(user)
	}
	return errors.New("UnfollowFunc not implemented")
}

func (m *mockGitHubClient) Follow(user string) error {
	if m.FollowFunc != nil {
		return m.FollowFunc(user)
	}
	return errors.New("FollowFunc not implemented")
}

func TestNewModel(t *testing.T) {
	m, ok := NewModel().(tuiModel)
	assert.True(t, ok)

	assert.NotNil(t, m.client)
	assert.True(t, m.loading)
	assert.Equal(t, followingPane, m.activePane)
	assert.Empty(t, m.followingList.Items())
	assert.Empty(t, m.followersList.Items())
}

func TestUpdate_TabKey(t *testing.T) {
	var m tea.Model = NewModel()
	var ok bool

	// First tab
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model, ok := m.(tuiModel)
	assert.True(t, ok)
	assert.Equal(t, followersPane, model.activePane)

	// Second tab
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model, ok = m.(tuiModel)
	assert.True(t, ok)
	assert.Equal(t, followingPane, model.activePane)
}

func TestUpdate_DataLoaded(t *testing.T) {
	var m tea.Model = NewModel()
	// m.loading is already true from NewModel()

	items := []list.Item{item("test1"), item("test2")}
	msg := dataLoadedMsg{
		username:      "testuser",
		onlyFollowing: items,
		onlyFollowers: items,
	}

	m, _ = m.Update(msg)
	updatedModel, ok := m.(tuiModel)
	assert.True(t, ok)

	assert.False(t, updatedModel.loading)
	assert.Nil(t, updatedModel.err)
	assert.Equal(t, "testuser", updatedModel.username)
	assert.Equal(t, items, updatedModel.followingList.Items())
	assert.Equal(t, items, updatedModel.followersList.Items())
}

func TestUpdate_Error(t *testing.T) {
	var m tea.Model = NewModel()
	// m.loading is already true from NewModel()

	expectedErr := errors.New("this is a test error")
	msg := errorMsg{err: expectedErr}

	m, _ = m.Update(msg)
	updatedModel, ok := m.(tuiModel)
	assert.True(t, ok)

	assert.False(t, updatedModel.loading)
	assert.Equal(t, expectedErr, updatedModel.err)
}
