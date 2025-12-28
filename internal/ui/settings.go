package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/llamasidekick/internal/config"
)

type settingsModel struct {
	cfg      *config.Config
	cursor   int
	settings []settingItem
}

type settingItem struct {
	name        string
	description string
	getValue    func(*config.Config) string
	toggle      func(*config.Config)
}

func (m settingsModel) Init() tea.Cmd {
	return nil
}

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc", "left", "h":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.settings)-1 {
				m.cursor++
			}

		case "enter", " ":
			// Toggle the setting
			if m.cursor < len(m.settings) {
				m.settings[m.cursor].toggle(m.cfg)
				// Save config
				if err := m.cfg.Save(); err != nil {
					fmt.Printf("\nError saving config: %v\n", err)
				}
			}
		}
	}

	return m, nil
}

func (m settingsModel) View() string {
	s := "\n\033[1;38;5;205m⚙️  Settings\033[0m\n\n"
	s += "\033[38;5;240mToggle settings with Enter or Space. Press 'q' to go back.\033[0m\n\n"

	for i, setting := range m.settings {
		cursor := " "
		if m.cursor == i {
			cursor = "\033[1;38;5;205m›\033[0m"
		}

		value := setting.getValue(m.cfg)
		s += fmt.Sprintf("%s \033[1m%s\033[0m: %s\n", cursor, setting.name, value)
		s += fmt.Sprintf("  \033[38;5;240m%s\033[0m\n\n", setting.description)
	}

	s += "\n\033[38;5;240mPress 'q' to go back\033[0m\n"

	return s
}

// RunSettings shows the settings menu
func RunSettings(cfg *config.Config) error {
	settings := []settingItem{
		{
			name:        "Debug Mode",
			description: "Show detailed request/response logs from Ollama",
			getValue: func(c *config.Config) string {
				if c.Ollama.Debug {
					return "\033[1;32mEnabled\033[0m"
				}
				return "\033[38;5;240mDisabled\033[0m"
			},
			toggle: func(c *config.Config) {
				c.Ollama.Debug = !c.Ollama.Debug
			},
		},
	}

	m := settingsModel{
		cfg:      cfg,
		cursor:   0,
		settings: settings,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
