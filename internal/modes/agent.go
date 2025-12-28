package modes

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/ollama"
	"github.com/yourusername/llamasidekick/internal/renderer"
	"github.com/yourusername/llamasidekick/internal/session"
)

// AgentMode provides autonomous task execution assistance
type AgentMode struct{}

func (m *AgentMode) Name() string {
	return "Agent"
}

func (m *AgentMode) Description() string {
	return "Autonomous multi-step task execution and problem solving"
}

func (m *AgentMode) GetSystemPrompt() string {
	return `You are an autonomous agent assistant capable of multi-step reasoning and task execution.

Your capabilities:
1. Break down complex tasks into actionable steps
2. Reason through problems systematically
3. Suggest sequences of actions to achieve goals
4. Identify prerequisites and dependencies
5. Anticipate potential issues and provide solutions

When given a task:
1. Analyze the requirements thoroughly
2. Create a step-by-step execution plan
3. Identify what information or tools are needed
4. Provide clear, actionable guidance
5. Think through potential obstacles

FORMATTING:
- Use markdown for clear communication
- Use bold (**text**) for emphasis
- Use headers (##) to organize sections
- Use numbered lists and bullet points
- Use code blocks with triple backticks when needed

Be thorough, methodical, and proactive in your assistance.`
}

func (m *AgentMode) Run(client *ollama.Client, sess *session.Session, cfg *config.Config) error {
	sess.SetMode(ModeAgent)
	
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("blue")).Render("\n=== AGENT MODE ==="))
	fmt.Println("Autonomous multi-step task execution and problem solving.")
	fmt.Println("Type 'exit' to return to main menu.\n")
	
	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Render("agent> "))
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		
		input = strings.TrimSpace(input)
		
		if input == "" {
			continue
		}
		
		if strings.ToLower(input) == "exit" {
			break
		}
		
		// Add user message to history
		sess.AddMessage("user", input)
		
		// Generate response
		fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("blue")).Render("\nAgent: "))
		
		var fullResponse strings.Builder
		modelName := cfg.GetModelForMode("agent")
		err = client.GenerateWithModel(
			modelName,
			input,
			m.GetSystemPrompt(),
			cfg.Ollama.Temperature,
			func(chunk string) error {
				fmt.Print(responseStyle.Render(chunk))
				fullResponse.WriteString(chunk)
				return nil
			},
		)
		
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			continue
		}
		
		// Render markdown
		markdown := fullResponse.String()
		renderedMd := renderer.RenderMarkdown(markdown)
		fmt.Print(renderedMd)
		fmt.Println("\n")
		
		// Add assistant response to history
		sess.AddMessage("assistant", markdown)
		
		// Save session
		if err := sess.Save(); err != nil {
			fmt.Printf("Warning: failed to save session: %v\n", err)
		}
	}
	
	return nil
}
