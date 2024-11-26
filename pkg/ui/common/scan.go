package common

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type ScanTickMsg struct{}

func ScanTick() tea.Cmd {
	return tea.Tick(time.Second/10, func(time.Time) tea.Msg {
		return ScanTickMsg{}
	})
}
