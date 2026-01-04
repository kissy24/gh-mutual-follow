package tui

import (
	"fmt"
	"sort"
	"time"

	"gh-mutual-follow/internal/github"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// clearStatusMsg clears the status message after a timeout.
func clearStatusMsg() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return statusMsg("")
	})
}

// loadDataCmd fetches all the necessary data from the GitHub client.
func loadDataCmd(client github.Client) tea.Cmd {
	return func() tea.Msg {
		username, err := client.GetUser()
		if err != nil {
			return errorMsg{fmt.Errorf("failed to get user: %w", err)}
		}

		following, err := client.GetFollowing(username)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to get following: %w", err)}
		}

		followers, err := client.GetFollowers(username)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to get followers: %w", err)}
		}

		onlyFollowingStr, onlyFollowersStr := github.GetMutualFollowsData(username, following, followers)

		// Create []item slices for sorting
		onlyFollowingItems := make([]item, len(onlyFollowingStr))
		for i, u := range onlyFollowingStr {
			onlyFollowingItems[i] = item(u)
		}
		sort.Slice(onlyFollowingItems, func(i, j int) bool {
			return onlyFollowingItems[i].FilterValue() < onlyFollowingItems[j].FilterValue()
		})

		onlyFollowersItems := make([]item, len(onlyFollowersStr))
		for i, u := range onlyFollowersStr {
			onlyFollowersItems[i] = item(u)
		}
		sort.Slice(onlyFollowersItems, func(i, j int) bool {
			return onlyFollowersItems[i].FilterValue() < onlyFollowersItems[j].FilterValue()
		})

		// Convert sorted []item to []list.Item for the message
		finalFollowingItems := make([]list.Item, len(onlyFollowingItems))
		for i, itm := range onlyFollowingItems {
			finalFollowingItems[i] = itm
		}

		finalFollowersItems := make([]list.Item, len(onlyFollowersItems))
		for i, itm := range onlyFollowersItems {
			finalFollowersItems[i] = itm
		}

		return dataLoadedMsg{
			username:      username,
			onlyFollowing: finalFollowingItems,
			onlyFollowers: finalFollowersItems,
		}
	}
}
