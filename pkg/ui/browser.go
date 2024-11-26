package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type entry struct {
	name  string
	isDir bool
}

type BrowserModel struct {
	currentPath  string
	entries      []entry
	cursor       int
	selectedFile string
	err          error
	viewport     viewport.Model
	ready        bool
	styles       struct {
		directory lipgloss.Style
		file      lipgloss.Style
		cursor    lipgloss.Style
		title     lipgloss.Style
	}
}

func NewBrowserModel() BrowserModel {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	m := BrowserModel{
		currentPath: home,
		cursor:      0,
	}
	m.styles.directory = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	m.styles.file = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	m.styles.cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	m.styles.title = lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		MarginBottom(1)
	m.loadEntries()
	return m
}

func (m *BrowserModel) Init() tea.Cmd {
	return nil
}

func (m *BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
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
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "backspace":
			if m.currentPath != "/" {
				m.currentPath = filepath.Dir(m.currentPath)
				m.loadEntries()
			}
		case "enter":
			selected := m.entries[m.cursor]
			fullPath := filepath.Join(m.currentPath, selected.name)
			if selected.isDir {
				m.currentPath = fullPath
				m.loadEntries()
			} else {
				m.selectedFile = fullPath
			}
		}
	}
	return *m, cmd
}

func (m *BrowserModel) SelectedFile() (bool, string) {
	return m.selectedFile != "", m.selectedFile
}

func (m *BrowserModel) ClearSelection() {
	m.selectedFile = ""
}

func (m BrowserModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var content string
	if m.err != nil {
		content = fmt.Sprintf("\nError: %v\n", m.err)
	} else {
		content = m.styles.title.Render(m.currentPath) + "\n\n"
		for i, entry := range m.entries {
			cursor := " "
			if i == m.cursor {
				cursor = m.styles.cursor.Render(">")
			}

			name := entry.name
			if entry.isDir {
				name = m.styles.directory.Render(name)
			} else {
				name = m.styles.file.Render(name)
			}

			content += fmt.Sprintf("%s %s\n", cursor, name)
		}
	}

	m.viewport.SetContent(content)
	return m.viewport.View()
}

func isAudioFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".mp3"
}

func (m *BrowserModel) loadEntries() {
	entries, err := os.ReadDir(m.currentPath)
	if err != nil {
		m.err = err
		return
	}

	var dirs, files []entry
	for _, e := range entries {
		name := e.Name()
		// Skip hidden files and directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		if e.IsDir() {
			dirs = append(dirs, entry{name: name, isDir: true})
		} else if isAudioFile(name) {
			files = append(files, entry{name: name, isDir: false})
		}
	}

	// Sort directories and files separately
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(
		files,
		func(i, j int) bool { return files[i].name < files[j].name },
	)

	// Combine directories first, then files
	m.entries = append(dirs, files...)
	m.cursor = 0
}