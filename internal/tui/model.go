package tui

import (
	"fmt"

	"gh-mutual-follow/internal/github"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	followingPane = iota
	followersPane
)

// model represents the state of the TUI.
type tuiModel struct {
	client                 github.Client
	username               string
	onlyFollowing          []list.Item
	onlyFollowers          []list.Item
	activePane             int
	followingList          list.Model
	followersList          list.Model
	loading                bool
	err                    error
	quitting               bool
	styles                 *TUIStyles
	statusMessage          string
	isBulkActionInProgress bool
	width, height          int
}

// NewModel creates the initial model for the TUI application.
func NewModel() tea.Model {
	styles := defaultStyles()
	client := github.NewClient()

	// Create delegates
	followingDelegate := itemDelegate{styles: styles}
	followersDelegate := itemDelegate{styles: styles}

	// Create lists
	followingList := list.New([]list.Item{}, followingDelegate, 0, 0)
	followersList := list.New([]list.Item{}, followersDelegate, 0, 0)

	followingList.SetShowTitle(false)
	followersList.SetShowTitle(false)
	followingList.KeyMap = list.DefaultKeyMap()
	followersList.KeyMap = list.DefaultKeyMap()
	followingList.Paginator.PerPage = 10
	followersList.Paginator.PerPage = 10

	return tuiModel{
		client:        client,
		activePane:    followingPane,
		followingList: followingList,
		followersList: followersList,
		loading:       true,
		styles:        styles,
	}
}

// Msgs for async operations
type dataLoadedMsg struct {
	username      string
	onlyFollowing []list.Item
	onlyFollowers []list.Item
	err           error
}

type errorMsg struct{ err error }

type statusMsg string

func (m tuiModel) Init() tea.Cmd {
	return loadDataCmd(m.client)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		const listHeight = 15 // Trial-and-error to get 10 items to display

		listWidth := msg.Width / 2

		m.followingList.SetHeight(listHeight)
		m.followersList.SetHeight(listHeight)
		m.followingList.SetWidth(listWidth)
		m.followersList.SetWidth(listWidth)
		return m, nil
	case dataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.username = msg.username
		m.onlyFollowing = msg.onlyFollowing
		m.onlyFollowers = msg.onlyFollowers

		m.followingList.SetItems(m.onlyFollowing)
		m.followersList.SetItems(m.onlyFollowers)

	case errorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case statusMsg:
		m.isBulkActionInProgress = false
		m.statusMessage = string(msg)
		if m.statusMessage != "" {
			return m, clearStatusMsg() // Start timer to clear message
		}
		return m, nil

	case tea.KeyMsg:
		if m.isBulkActionInProgress {
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		if m.err != nil {
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		var cmd tea.Cmd
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "tab", "shift+tab":
			if m.activePane == followingPane {
				m.activePane = followersPane
			} else {
				m.activePane = followingPane
			}
			return m, nil
		case "r":
			m.loading = true
			m.err = nil
			return m, loadDataCmd(m.client)
		case "enter":
			var selectedItem item
			var actionCmd tea.Cmd

			if m.activePane == followingPane {
				if i := m.followingList.SelectedItem(); i != nil {
					selectedItem = i.(item)
					actionCmd = func() tea.Msg {
						err := m.client.Unfollow(string(selectedItem))
						if err != nil {
							return errorMsg{fmt.Errorf("failed to unfollow %s: %w", selectedItem, err)}
						}
						return statusMsg(fmt.Sprintf("Unfollowed %s!", selectedItem))
					}
				}
			} else { // Followers pane
				if i := m.followersList.SelectedItem(); i != nil {
					selectedItem = i.(item)
					actionCmd = func() tea.Msg {
						err := m.client.Follow(string(selectedItem))
						if err != nil {
							return errorMsg{fmt.Errorf("failed to follow %s: %w", selectedItem, err)}
						}
						return statusMsg(fmt.Sprintf("Followed %s!", selectedItem))
					}
				}
			}

			if actionCmd != nil {
				m.loading = true
				return m, tea.Batch(actionCmd, loadDataCmd(m.client))
			}
		case "a": // Bulk action
			var items []list.Item
			var action string
			if m.activePane == followingPane {
				items = m.followingList.Items()
				action = "unfollow"
			} else {
				items = m.followersList.Items()
				action = "follow"
			}

			if len(items) == 0 {
				return m, nil
			}

			m.isBulkActionInProgress = true
			m.statusMessage = fmt.Sprintf("Bulk %sing all users...", action)

			return m, func() tea.Msg {
				for _, i := range items {
					user := i.(item)
					if action == "unfollow" {
						_ = m.client.Unfollow(string(user)) // Errors are ignored for now in bulk action
					} else {
						_ = m.client.Follow(string(user))
					}
				}
				m.isBulkActionInProgress = false // Reset after completion
				return statusMsg(fmt.Sprintf("Bulk %s complete!", action))
			}
		default: // Forward other keys (like arrows) to the active list
			if m.activePane == followingPane {
				m.followingList, cmd = m.followingList.Update(msg)
			} else {
				m.followersList, cmd = m.followersList.Update(msg)
			}
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		return m.styles.LoadingStyle.Render("Loading data...") + "\n"
	}

	if m.err != nil {
		return m.styles.ErrorStyle.Render("Error: " + m.err.Error()) + "\n" +
			m.styles.HelpStyle.Render("[q] to quit") + "\n"
	}

	headerView := m.styles.Header.Width(m.width).Render(fmt.Sprintf("GitHub Account : %s", m.username))
	helpView := m.styles.HelpStyle.Render("[q] Quit   [↑↓] Move   [←→] Page   [tab] Switch Pane   [r] Refresh   [enter] Action   [a] Action All")
	statusView := ""
	if m.isBulkActionInProgress {
		statusView = m.styles.StatusMessage.Render("Working...")
	} else if m.statusMessage != "" {
		statusView = m.styles.StatusMessage.Render(m.statusMessage)
	}

	footerView := lipgloss.JoinVertical(lipgloss.Left, helpView, statusView)

	// Render panes
	followingTitle := "Following"
	followersTitle := "Followers"

	followingContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(followingTitle),
		m.followingList.View(),
	)
	followersContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(followersTitle),
		m.followersList.View(),
	)

	var leftPane, rightPane string
	if m.activePane == followingPane {
		leftPane = m.styles.FocusedPane.Render(followingContent)
		rightPane = m.styles.Pane.Render(followersContent)
	} else {
		leftPane = m.styles.Pane.Render(followingContent)
		rightPane = m.styles.FocusedPane.Render(followersContent)
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return lipgloss.JoinVertical(lipgloss.Left,
		headerView,
		content,
		footerView,
	)
}
