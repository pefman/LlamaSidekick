package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/ollama"
)

type firstRunModel struct {
	client          *ollama.Client
	cfg             *config.Config
	availableModels []ollama.Model
	cursor          int
	selected        bool
	err             error
	loading         bool
}

func newFirstRunModel(client *ollama.Client, cfg *config.Config) firstRunModel {
	return firstRunModel{
		client:  client,
		cfg:     cfg,
		cursor:  0,
		loading: true,
	}
}

func (m firstRunModel) Init() tea.Cmd {
	return func() tea.Msg {
		models, err := m.client.ListModels()
		if err != nil {
			return errMsg{err}
		}
		return modelsLoadedMsg{models}
	}
}

func (m firstRunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case modelsLoadedMsg:
		m.availableModels = msg.models
		m.loading = false
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.availableModels)-1 {
				m.cursor++
			}

		case "enter":
			if len(m.availableModels) > 0 {
				m.selected = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m firstRunModel) View() string {
	var s strings.Builder

	// Title - bold + magenta
	s.WriteString("\n\033[1;38;5;205m🦙 Welcome to LlamaSidekick!\033[0m\n\n")

	if m.err != nil {
		s.WriteString("\033[38;5;9mError: " + m.err.Error() + "\033[0m\n\n")
		s.WriteString("\033[38;5;240mPress q to quit\033[0m\n")
		return s.String()
	}

	if m.loading {
		s.WriteString("\033[38;5;240mDetecting available Ollama models...\033[0m\n")
		return s.String()
	}

	if len(m.availableModels) == 0 {
		s.WriteString("\033[38;5;9mNo Ollama models found!\033[0m\n\n")
		s.WriteString("\033[38;5;240mPlease install a model first with: ollama pull codellama\033[0m\n")
		s.WriteString("\033[38;5;240mPress q to quit\033[0m\n")
		return s.String()
	}

	s.WriteString(fmt.Sprintf("\033[38;5;240mFound %d model(s). Select a default model:\033[0m\n\n", len(m.availableModels)))

	for i, model := range m.availableModels {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		size := float64(model.Size) / (1024 * 1024 * 1024)
		sizeStr := fmt.Sprintf("%.1f GB", size)

		// Use ANSI codes directly to avoid lipgloss alignment issues
		if m.cursor == i {
			// Bold + color for selected item
			s.WriteString(cursor + "\033[1;38;5;170m" + model.Name + "\033[0m\n")
		} else {
			s.WriteString(cursor + model.Name + "\n")
		}
		s.WriteString("  " + sizeStr + "\n")
	}

	s.WriteString("\n")
	s.WriteString("\033[38;5;240mThis model will be used for all modes by default.\033[0m\n")
	s.WriteString("\033[38;5;240mYou can configure different models per mode later via 'Configure Models'.\033[0m\n\n")
	s.WriteString("\033[38;5;240mPress Enter to select, q to quit\033[0m\n")

	return s.String()
}

// RunFirstRun shows the first-run model selection and returns the selected model
func RunFirstRun(client *ollama.Client, cfg *config.Config) (string, error) {
	p := tea.NewProgram(newFirstRunModel(client, cfg), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	model := m.(firstRunModel)
	if model.err != nil {
		return "", model.err
	}

	if !model.selected || len(model.availableModels) == 0 {
		return "", fmt.Errorf("no model selected")
	}

	return model.availableModels[model.cursor].Name, nil
}
