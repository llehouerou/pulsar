package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/llehouerou/pulsar/pkg/player"
)

type PlayerModel struct {
	player       *player.Player
	playing      bool
	err          error
	viewport     viewport.Model
	ready        bool
	progress     progress.Model
	showTimeLeft bool
	styles       struct {
		status   lipgloss.Style
		help     lipgloss.Style
		metadata lipgloss.Style
		time     lipgloss.Style
	}
}

type tickMsg time.Time

func NewPlayerModel() PlayerModel {
	m := PlayerModel{
		player:       player.New(),
		playing:      false,
		showTimeLeft: false,
		progress: progress.New(
			progress.WithScaledGradient("#FF7CCB", "#FDFF8C"),
		),
	}
	m.styles.status = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	m.styles.help = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	m.styles.metadata = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))
	m.styles.time = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	return m
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d", m, s)
}

func (m *PlayerModel) Update(msg tea.Msg) (PlayerModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.viewport.Style = lipgloss.NewStyle().Align(lipgloss.Center)
			m.progress.Width = msg.Width - 20
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
			m.progress.Width = msg.Width - 20
		}
	case playerErrorMsg:
		m.err = msg.error
	case playerStartedMsg:
		m.playing = true
		return *m, tickCmd()
	case tickMsg:
		if m.playing {
			return *m, tickCmd()
		}
	case tea.KeyMsg:
		switch msg.String() {
		case " ": // Space key
			m.player.Toggle()
			m.playing = !m.playing
			if m.playing {
				return *m, tickCmd()
			}
			return *m, nil
		case "t": // Toggle time display
			m.showTimeLeft = !m.showTimeLeft
		case "esc":
			return *m, nil
		case "ctrl+c", "q":
			return *m, tea.Quit
		}
	}
	return *m, cmd
}

func (m *PlayerModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var content string
	if m.err != nil {
		content = fmt.Sprintf("\nError: %v\n", m.err)
	} else {
		metadata := m.player.GetMetadata()
		status := " Paused"
		if m.playing {
			status = " Playing"
		}

		// Center each section
		centerStyle := lipgloss.NewStyle().Width(m.viewport.Width).Align(lipgloss.Center)

		// Status
		content = centerStyle.Render(m.styles.status.Render(status)) + "\n"

		// Metadata
		if metadata.Artist != "" || metadata.Title != "" {
			content += centerStyle.Render(
				m.styles.metadata.Render(
					fmt.Sprintf("%s - %s",
						metadata.Artist,
						metadata.Title,
					),
				),
			) + "\n\n"
		}

		// Progress bar
		content += centerStyle.Render(m.progress.ViewAs(m.player.Position())) + "\n"

		// Time display
		position := m.player.CurrentPosition()
		duration := m.player.Duration()
		timeDisplay := formatDuration(position)
		if m.showTimeLeft {
			timeDisplay += " / -" + formatDuration(duration-position)
		} else {
			timeDisplay += " / " + formatDuration(duration)
		}
		content += centerStyle.Render(m.styles.time.Render(timeDisplay)) + "\n\n"

		// Help text
		helpText := m.styles.help.Render(strings.Join([]string{
			"Space: Play/Pause",
			"t: Toggle time display",
			"Esc: Back to browser",
			"q: Quit",
		}, "\n"))
		content += centerStyle.Render(helpText)
	}

	m.viewport.SetContent(content)
	return m.viewport.View()
}

func (m *PlayerModel) StartPlayback(filepath string) tea.Cmd {
	return func() tea.Msg {
		err := m.player.Play(filepath)
		if err != nil {
			return playerErrorMsg{err}
		}
		return playerStartedMsg{}
	}
}

func (m *PlayerModel) Stop() {
	if m.player != nil {
		m.player.Stop()
		m.player.Close()
	}
	m.playing = false
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type playerErrorMsg struct {
	error
}

type playerStartedMsg struct{}
