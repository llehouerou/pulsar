package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/llehouerou/pulsar/pkg/ui"
)

func main() {
	p := tea.NewProgram(ui.NewModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
