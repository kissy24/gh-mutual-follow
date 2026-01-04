package tui

import "github.com/charmbracelet/lipgloss"

// TUIStyles holds the styles for the TUI.
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
