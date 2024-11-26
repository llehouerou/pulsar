package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/llehouerou/pulsar/pkg/db"
	"github.com/llehouerou/pulsar/pkg/ui"
)

func main() {
	// Get config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Printf("Error getting config directory: %v\n", err)
		os.Exit(1)
	}

	// Create pulsar config directory if it doesn't exist
	pulsarDir := filepath.Join(configDir, "pulsar")
	if err := os.MkdirAll(pulsarDir, 0755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize database
	database, err := db.New(filepath.Join(pulsarDir, "pulsar.db"))
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	p := tea.NewProgram(ui.NewModel(database))

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
