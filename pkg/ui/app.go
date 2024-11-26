package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/llehouerou/pulsar/pkg/db"
	"github.com/llehouerou/pulsar/pkg/media"
)

type Screen int

const (
	BrowserScreen Screen = iota
	PlayerScreen
	AddSourceScreen
)

type Model struct {
	currentScreen Screen
	browser       BrowserModel
	player        PlayerModel
	addSource     AddSourceModel
	manager       *media.SourceManager
}

func NewModel(database *db.DB) Model {
	manager := media.NewSourceManager(database)
	// Register filesystem source type
	manager.RegisterSourceType("filesystem", media.NewFilesystemSourceFactory())
	// Load existing sources
	if err := manager.LoadSources(); err != nil {
		panic(err)
	}

	return Model{
		currentScreen: BrowserScreen,
		browser:       NewBrowserModel(manager),
		player:        NewPlayerModel(),
		addSource:     NewAddSourceModel(manager),
		manager:       manager,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		// Update all models with window size
		m.browser, _ = m.browser.Update(msg)
		m.player, _ = m.player.Update(msg)
		m.addSource, _ = m.addSource.Update(msg)
	}

	switch m.currentScreen {
	case BrowserScreen:
		return m.updateBrowser(msg)
	case PlayerScreen:
		return m.updatePlayer(msg)
	case AddSourceScreen:
		return m.updateAddSource(msg)
	}
	return m, cmd
}

func (m Model) updateBrowser(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// First handle the browser update
	m.browser, cmd = m.browser.Update(msg)

	// Then check if a file was selected
	if selected, path := m.browser.SelectedFile(); selected {
		if path == "ADD_SOURCE" {
			m.currentScreen = AddSourceScreen
			m.browser.ClearSelection()
			return m, cmd
		}

		// Stop the current player before starting a new one
		m.player.Stop()
		m.currentScreen = PlayerScreen
		m.player = NewPlayerModel()
		// Initialize the new player with the current window size
		if m.browser.ready {
			m.player, _ = m.player.Update(tea.WindowSizeMsg{
				Width:  m.browser.viewport.Width,
				Height: m.browser.viewport.Height,
			})
		}
		// Clear the selection so we don't keep triggering it
		m.browser.ClearSelection()
		// Return a batch of commands - both the browser update and starting playback
		return m, tea.Batch(cmd, m.player.StartPlayback(path))
	}

	// Handle other key events
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, cmd
}

func (m Model) updatePlayer(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.currentScreen = BrowserScreen
			return m, nil
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	m.player, cmd = m.player.Update(msg)
	return m, cmd
}

func (m Model) updateAddSource(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.addSource, cmd = m.addSource.Update(msg)

	if m.addSource.Done() {
		m.currentScreen = BrowserScreen
		// Create a new browser model to refresh the sources
		m.browser = NewBrowserModel(m.manager)
		// Initialize the browser with the current window size
		if m.addSource.ready {
			m.browser, _ = m.browser.Update(tea.WindowSizeMsg{
				Width:  m.addSource.viewport.Width,
				Height: m.addSource.viewport.Height,
			})
		}
	}

	return m, cmd
}

func (m Model) View() string {
	switch m.currentScreen {
	case BrowserScreen:
		return m.browser.View()
	case PlayerScreen:
		return m.player.View()
	case AddSourceScreen:
		return m.addSource.View()
	default:
		return "Unknown screen"
	}
}
