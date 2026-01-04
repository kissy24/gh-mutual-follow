package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// item represents an item in the list.
type item string

// FilterValue is required by the list.Model interface.
func (i item) FilterValue() string { return string(i) }

// itemDelegate is responsible for rendering list items.
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
