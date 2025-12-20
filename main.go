package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gh-mutual-follow/internal/cli"
)

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

const (
	followingPane = iota
	followersPane
)

type item string

func (i item) FilterValue() string { return string(i) }

type model struct {
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

type TUIStyles struct {
	Header        lipgloss.Style
	Pane          lipgloss.Style
	FocusedPane   lipgloss.Style
	HelpStyle     lipgloss.Style
	CursorStyle   lipgloss.Style
	SelectedStyle lipgloss.Style
	NoItemsStyle  lipgloss.Style
	LoadingStyle  lipgloss.Style
	ErrorStyle    lipgloss.Style
	StatusMessage lipgloss.Style
}

func defaultStyles() *TUIStyles {
	s := new(TUIStyles)
	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		PaddingLeft(1).
		PaddingRight(1)

	s.Pane = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1).
		Height(16)

	s.FocusedPane = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1).
		Height(16)

	s.HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(1).PaddingRight(1)
	s.CursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	s.SelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
	s.NoItemsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	s.LoadingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
	s.ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	s.StatusMessage = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(1)

	return s
}

// custom itemDelegate to style items
type itemDelegate struct {
	styles *TUIStyles
}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	i, ok := li.(item)
	if !ok {
		return
	}

	str := i.FilterValue()

	if index == m.Index() {
		fmt.Fprintf(w, "%s%s%s", d.styles.CursorStyle.Render("> "), d.styles.SelectedStyle.Render(str), "\n")
	} else {
		fmt.Fprintf(w, "  %s\n", str)
	}
}

func initialModel() model {
	styles := defaultStyles()

	// Create delegates
	followingDelegate := itemDelegate{styles: styles}
	followersDelegate := itemDelegate{styles: styles}

	// Create lists
	followingList := list.New([]list.Item{}, followingDelegate, 0, 0)
	followersList := list.New([]list.Item{}, followersDelegate, 0, 0)

	// Configure lists
	followingList.SetShowTitle(false) // We'll render the title manually
	followersList.SetShowTitle(false)

	followingList.KeyMap = list.DefaultKeyMap()
	followersList.KeyMap = list.DefaultKeyMap()

	followingList.Paginator.PerPage = 10
	followersList.Paginator.PerPage = 10

	return model{
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

type statusMsg string // New message type for transient status messages

func clearStatusMsg() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return statusMsg("")
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadDataCmd())
}

func (m model) loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		username, err := cli.GetUser()
		if err != nil {
			return errorMsg{fmt.Errorf("failed to get user: %w", err)}
		}

		following, err := cli.GetFollowing(username)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to get following: %w", err)}
		}

		followers, err := cli.GetFollowers(username)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to get followers: %w", err)}
		}

		onlyFollowingStr, onlyFollowersStr := cli.GetMutualFollowsData(username, following, followers)

		var onlyFollowingItems []list.Item
		for _, u := range onlyFollowingStr {
			onlyFollowingItems = append(onlyFollowingItems, item(u))
		}

		var onlyFollowersItems []list.Item
		for _, u := range onlyFollowersStr {
			onlyFollowersItems = append(onlyFollowersItems, item(u))
		}

		sort.Slice(onlyFollowingItems, func(i, j int) bool {
			return onlyFollowingItems[i].FilterValue() < onlyFollowingItems[j].FilterValue()
		})
		sort.Slice(onlyFollowersItems, func(i, j int) bool {
			return onlyFollowersItems[i].FilterValue() < onlyFollowersItems[j].FilterValue()
		})

		return dataLoadedMsg{
			username:      username,
			onlyFollowing: onlyFollowingItems,
			onlyFollowers: onlyFollowersItems,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case statusMsg: // Handle status messages
		m.isBulkActionInProgress = false
		m.statusMessage = string(msg)
		if m.statusMessage != "" {
			return m, clearStatusMsg() // Start timer to clear message
		}
		return m, nil

	case tea.KeyMsg:
		if m.isBulkActionInProgress { // If a bulk action is in progress, ignore other key presses
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		if m.err != nil { // If there's an error, only allow quitting
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
		case "tab", "shift+tab": // Switch panes
			if m.activePane == followingPane {
				m.activePane = followersPane
			} else {
				m.activePane = followingPane
			}
			return m, nil
		case "r": // Refresh data
			m.loading = true
			m.err = nil
			return m, m.loadDataCmd()
		case "enter": // Perform action on selected item
			var selectedItem item
			var actionCmd tea.Cmd

			if m.activePane == followingPane {
				if i := m.followingList.SelectedItem(); i != nil {
					selectedItem = i.(item)
					actionCmd = func() tea.Msg {
						err := cli.Unfollow(string(selectedItem))
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
						err := cli.Follow(string(selectedItem))
						if err != nil {
							return errorMsg{fmt.Errorf("failed to follow %s: %w", selectedItem, err)}
						}
						return statusMsg(fmt.Sprintf("Followed %s!", selectedItem))
					}
				}
			}

			if actionCmd != nil {
				m.loading = true
				return m, tea.Batch(actionCmd, m.loadDataCmd())
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
						_ = cli.Unfollow(string(user)) // Errors are ignored for now in bulk action
					} else {
						_ = cli.Follow(string(user))
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

func (m model) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		return m.styles.LoadingStyle.Render("Loading data...") + "\n"
	}

	if m.err != nil {
		return m.styles.ErrorStyle.Render("Error: "+m.err.Error()) + "\n" +
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

	// Create content for panes
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
