package main

import (
	"fmt"
	"os"

	"gh-mutual-follow/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := tui.NewModel()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
