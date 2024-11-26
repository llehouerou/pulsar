package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/llehouerou/pulsar/pkg/player"
)

type PlayerModel struct {
	player   *player.Player
	playing  bool
	err      error
	viewport viewport.Model
	ready    bool
	styles   struct {
		status lipgloss.Style
		help   lipgloss.Style
	}
}

func NewPlayerModel() PlayerModel {
	m := PlayerModel{
		player:  player.New(),
		playing: false,
	}
	m.styles.status = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	m.styles.help = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	return m
}

func (m PlayerModel) Update(msg tea.Msg) (PlayerModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.viewport.Style = lipgloss.NewStyle().Align(lipgloss.Center)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}
	case playerErrorMsg:
		m.err = msg.error
	case playerStartedMsg:
		m.playing = true
	case tea.KeyMsg:
		switch msg.String() {
		case " ": // Space key
			m.player.Toggle()
			m.playing = !m.playing
			return m, nil
		case "esc":
			return m, nil
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, cmd
}

func (m PlayerModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var content string
	if m.err != nil {
		content = fmt.Sprintf("\nError: %v\n", m.err)
	} else {
		status := " Paused"
		if m.playing {
			status = " Playing"
		}
		content = m.styles.status.Render(status) + "\n\n" +
			m.styles.help.Render("Space: Play/Pause\n"+
				"Esc: Back to browser\n"+
				"q: Quit")
	}

	m.viewport.SetContent(content)
	return m.viewport.View()
}

func (m PlayerModel) StartPlayback(filepath string) tea.Cmd {
	return func() tea.Msg {
		err := m.player.Play(filepath)
		if err != nil {
			return playerErrorMsg{err}
		}
		m.playing = true
		return playerStartedMsg{}
	}
}

func (m PlayerModel) Stop() {
	if m.player != nil {
		m.player.Stop()
		m.player.Close()
	}
	m.playing = false
}

type playerErrorMsg struct {
	error
}

type playerStartedMsg struct{}
