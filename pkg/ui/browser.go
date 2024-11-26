package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/llehouerou/pulsar/pkg/media"
	"github.com/llehouerou/pulsar/pkg/ui/common"
)

type BrowserMode int

const (
	SourcesMode BrowserMode = iota
	TracksMode
)

const (
	scrollMargin = 3 // Number of lines to keep as margin at top and bottom
)

type BrowserModel struct {
	mode          BrowserMode
	sources       []media.SourceConfig
	tracks        []media.Track
	currentSource string
	sourceCursor  int
	trackCursor   int
	selectedTrack string
	err           error
	viewport      viewport.Model
	ready         bool
	manager       *media.SourceManager
	progress      progress.Model
	scanning      bool
	styles        struct {
		title    lipgloss.Style
		source   lipgloss.Style
		track    lipgloss.Style
		cursor   lipgloss.Style
		metadata lipgloss.Style
		progress lipgloss.Style
		status   lipgloss.Style
	}
}

type scanTickMsg struct{}

func scanTick() tea.Cmd {
	return tea.Tick(time.Second/10, func(time.Time) tea.Msg {
		return scanTickMsg{}
	})
}

func NewBrowserModel(manager *media.SourceManager) BrowserModel {
	m := BrowserModel{
		mode:         SourcesMode,
		sourceCursor: 0,
		trackCursor:  0,
		manager:      manager,
		progress: progress.New(
			progress.WithScaledGradient("#FF7CCB", "#FDFF8C"),
		),
	}

	m.styles.title = lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		MarginBottom(1)
	m.styles.source = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	m.styles.track = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	m.styles.cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	m.styles.metadata = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	m.styles.progress = lipgloss.NewStyle().MarginTop(1)
	m.styles.status = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	m.loadSources()
	return m
}

func (m *BrowserModel) loadSources() {
	m.sources = m.manager.GetSources()
	m.sourceCursor = 0
}

func (m *BrowserModel) loadTracks() error {
	tracks, err := m.manager.GetTracks(m.currentSource)
	if err != nil {
		return err
	}
	m.tracks = tracks
	m.trackCursor = 0
	return nil
}

func (m *BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 4 // Title + spacing
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight)
			m.viewport.Style = lipgloss.NewStyle().Align(lipgloss.Center)
			m.progress.Width = msg.Width - 20
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight
			m.progress.Width = msg.Width - 20
		}

	case common.ScanTickMsg:
		if m.scanning {
			if progress := m.manager.GetScanProgress(); progress != nil {
				return *m, common.ScanTick()
			}
			// Scanning finished
			m.scanning = false
			if err := m.loadTracks(); err != nil {
				m.err = err
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			switch m.mode {
			case SourcesMode:
				if m.sourceCursor > 0 {
					m.sourceCursor--
					// Update viewport position
					if m.sourceCursor < m.viewport.YOffset+scrollMargin {
						m.viewport.YOffset = max(0, m.sourceCursor-scrollMargin)
					}
				}
			case TracksMode:
				if m.trackCursor > 0 {
					m.trackCursor--
					// Update viewport position
					if m.trackCursor < m.viewport.YOffset+scrollMargin {
						m.viewport.YOffset = max(0, m.trackCursor-scrollMargin)
					}
				}
			}
		case "down":
			switch m.mode {
			case SourcesMode:
				if m.sourceCursor < len(m.sources)-1 {
					m.sourceCursor++
					// Update viewport position
					if m.sourceCursor >= m.viewport.YOffset+m.viewport.Height-scrollMargin {
						maxOffset := max(0, len(m.sources)-m.viewport.Height+scrollMargin)
						m.viewport.YOffset = min(
							m.sourceCursor-m.viewport.Height+1+scrollMargin,
							maxOffset,
						)
					}
				}
			case TracksMode:
				if m.trackCursor < len(m.tracks)-1 {
					m.trackCursor++
					// Update viewport position
					if m.trackCursor >= m.viewport.YOffset+m.viewport.Height-scrollMargin {
						maxOffset := max(0, len(m.tracks)-m.viewport.Height+scrollMargin)
						m.viewport.YOffset = min(
							m.trackCursor-m.viewport.Height+1+scrollMargin,
							maxOffset,
						)
					}
				}
			}
		case "backspace", "esc":
			if m.mode == TracksMode {
				m.mode = SourcesMode
				m.currentSource = ""
				m.viewport.YOffset = 0
			}
		case "enter":
			switch m.mode {
			case SourcesMode:
				if len(m.sources) > 0 && m.sourceCursor < len(m.sources) {
					m.currentSource = m.sources[m.sourceCursor].ID
					m.mode = TracksMode
					m.viewport.YOffset = 0
					if err := m.loadTracks(); err != nil {
						m.err = err
					}
				}
			case TracksMode:
				if len(m.tracks) > 0 && m.trackCursor < len(m.tracks) {
					m.selectedTrack = m.tracks[m.trackCursor].Path
				}
			}
		case "a":
			if m.mode == SourcesMode {
				m.selectedTrack = "ADD_SOURCE"
			}
		case "r":
			if m.mode == TracksMode && !m.scanning {
				m.scanning = true
				go m.manager.ScanSource(context.Background(), m.currentSource)
				return *m, common.ScanTick()
			}
		}
	}
	return *m, cmd
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m BrowserModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var content string
	if m.err != nil {
		content = fmt.Sprintf("\nError: %v\n", m.err)
	} else {
		title := ""
		switch m.mode {
		case SourcesMode:
			title = "Music Sources"
			var list strings.Builder
			for i, source := range m.sources {
				cursor := " "
				if i == m.sourceCursor {
					cursor = m.styles.cursor.Render(">")
				}
				name := m.styles.source.Render(source.Name)
				list.WriteString(fmt.Sprintf("%s %s\n", cursor, name))
			}
			if len(m.sources) == 0 {
				list.WriteString("No sources configured. Press 'a' to add a source.")
			}
			content = list.String()

		case TracksMode:
			if m.sourceCursor >= len(m.sources) {
				content = "No source selected."
				break
			}

			source := m.sources[m.sourceCursor]
			title = source.Name

			var list strings.Builder
			// Show scanning progress if active
			if progress := m.manager.GetScanProgress(); progress != nil && progress.SourceID == m.currentSource {
				var percent float64
				if progress.Total > 0 {
					percent = float64(progress.Current) / float64(progress.Total)
				}
				list.WriteString(m.styles.progress.Render(m.progress.ViewAs(percent)) + "\n")
				list.WriteString(m.styles.status.Render(
					fmt.Sprintf(
						"%s (%d/%d files)\n\n",
						progress.Status,
						progress.Current,
						progress.Total,
					),
				))
			}

			for i, track := range m.tracks {
				cursor := " "
				if i == m.trackCursor {
					cursor = m.styles.cursor.Render(">")
				}
				title := track.Title
				if title == "" {
					title = "Unknown Title"
				}
				artist := track.Artist
				if artist == "" {
					artist = "Unknown Artist"
				}

				trackInfo := m.styles.track.Render(title)
				metadata := m.styles.metadata.Render(fmt.Sprintf(" - %s", artist))
				list.WriteString(fmt.Sprintf("%s %s%s\n", cursor, trackInfo, metadata))
			}
			if len(m.tracks) == 0 {
				list.WriteString("No tracks found. Press 'r' to rescan.")
			}
			content = list.String()
		}

		// Add title above viewport
		content = m.styles.title.Render(title) + "\n\n" + content
	}

	m.viewport.SetContent(content)
	return m.viewport.View()
}

func (m *BrowserModel) SelectedFile() (bool, string) {
	return m.selectedTrack != "", m.selectedTrack
}

func (m *BrowserModel) ClearSelection() {
	m.selectedTrack = ""
}
