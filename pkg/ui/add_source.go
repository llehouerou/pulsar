package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/llehouerou/pulsar/pkg/media"
	"github.com/llehouerou/pulsar/pkg/ui/common"
)

type AddSourceModel struct {
	nameInput  textinput.Model
	pathsInput textinput.Model
	viewport   viewport.Model
	ready      bool
	done       bool
	err        error
	manager    *media.SourceManager
	scanning   bool
	styles     struct {
		title lipgloss.Style
		label lipgloss.Style
		error lipgloss.Style
		help  lipgloss.Style
	}
}

func NewAddSourceModel(manager *media.SourceManager) AddSourceModel {
	m := AddSourceModel{
		nameInput:  textinput.New(),
		pathsInput: textinput.New(),
		manager:    manager,
	}

	m.nameInput.Placeholder = "My Music Collection"
	m.nameInput.Focus()
	m.pathsInput.Placeholder = "/path/to/music;/another/path"

	m.styles.title = lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		MarginBottom(1)
	m.styles.label = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	m.styles.error = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	m.styles.help = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	return m
}

func (m *AddSourceModel) Update(msg tea.Msg) (AddSourceModel, tea.Cmd) {
	var cmds []tea.Cmd

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

	case common.ScanTickMsg:
		if m.scanning {
			if progress := m.manager.GetScanProgress(); progress != nil {
				var content strings.Builder
				content.WriteString(m.styles.title.Render("Adding Music Source") + "\n\n")
				content.WriteString(fmt.Sprintf("Scanning files: %d found\n", progress.Current))
				m.viewport.SetContent(content.String())
				return *m, common.ScanTick()
			}
			// Scanning finished
			m.scanning = false
			m.done = true
		}

	case tea.KeyMsg:
		if m.scanning {
			return *m, nil // Ignore key presses while scanning
		}

		switch msg.String() {
		case "tab", "shift+tab", "enter", "up", "down":
			// Cycle between inputs
			if m.nameInput.Focused() {
				m.nameInput.Blur()
				m.pathsInput.Focus()
			} else {
				m.nameInput.Focus()
				m.pathsInput.Blur()
			}
			return *m, nil

		case "esc":
			m.done = true
			return *m, nil

		case "ctrl+s":
			if m.nameInput.Value() == "" {
				m.err = fmt.Errorf("name is required")
				return *m, nil
			}
			if m.pathsInput.Value() == "" {
				m.err = fmt.Errorf("at least one path is required")
				return *m, nil
			}

			m.err = nil
			m.scanning = true

			// Add source in a goroutine to avoid blocking the UI
			go func() {
				err := m.manager.AddSource(
					m.nameInput.Value(),
					"filesystem",
					map[string]string{
						"paths": m.pathsInput.Value(),
					},
				)
				if err != nil {
					m.err = err
					m.scanning = false
				}
			}()
			return *m, common.ScanTick()
		}
	}

	// Handle character input
	if m.nameInput.Focused() {
		newNameInput, cmd := m.nameInput.Update(msg)
		m.nameInput = newNameInput
		cmds = append(cmds, cmd)
	} else {
		newPathsInput, cmd := m.pathsInput.Update(msg)
		m.pathsInput = newPathsInput
		cmds = append(cmds, cmd)
	}

	return *m, tea.Batch(cmds...)
}

func (m AddSourceModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var content strings.Builder
	content.WriteString(m.styles.title.Render("Add Music Source") + "\n\n")

	if m.scanning {
		if progress := m.manager.GetScanProgress(); progress != nil {
			content.WriteString(
				fmt.Sprintf("Scanning files: %d found\n", progress.Current),
			)
		} else {
			content.WriteString("Starting scan...\n")
		}
		m.viewport.SetContent(content.String())
		return m.viewport.View()
	}

	// Name input
	content.WriteString(m.styles.label.Render("Name:") + "\n")
	content.WriteString(m.nameInput.View() + "\n\n")

	// Paths input
	content.WriteString(
		m.styles.label.Render("Music Paths (semicolon-separated):") + "\n",
	)
	content.WriteString(m.pathsInput.View() + "\n\n")

	// Error message
	if m.err != nil {
		content.WriteString(m.styles.error.Render(m.err.Error()) + "\n\n")
	}

	// Help text
	content.WriteString(m.styles.help.Render(
		"tab: Switch fields • ctrl+s: Save • esc: Cancel",
	))

	m.viewport.SetContent(content.String())
	return m.viewport.View()
}

func (m AddSourceModel) Done() bool {
	return m.done
}
