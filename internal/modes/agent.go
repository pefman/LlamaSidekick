package modes

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
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
6. CREATE FILES automatically when generating scripts or code

When given a task:
1. Analyze the requirements thoroughly
2. Create a step-by-step execution plan
3. Identify what information or tools are needed
4. Provide clear, actionable guidance
5. Think through potential obstacles
6. When you provide a script or code, specify the filename using this format:
   FILENAME: path/to/file.ext
   Followed immediately by the code block with triple backticks

FORMATTING:
- Use markdown for clear communication
- Use bold (**text**) for emphasis
- Use headers (##) to organize sections
- Use numbered lists and bullet points
- CRITICAL: When providing code/scripts, use this exact format:
  FILENAME: script_name.sh
  Then add a code block with the language specified (e.g., bash, python, go)
  The file will be automatically created with the code content

Be thorough, methodical, and proactive in your assistance. CREATE files automatically.`
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
		
		// Start spinner
		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Suffix = " Thinking..."
		s.Start()
		
		var fullResponse strings.Builder
		modelName := cfg.GetModelForMode("agent")
		err = client.GenerateWithModel(
			modelName,
			input,
			m.GetSystemPrompt(),
			cfg.Ollama.Temperature,
			func(chunk string) error {
				if s.Active() {
					s.Stop()
					fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("blue")).Render("\nAgent: "))
				}
				fmt.Print(responseStyle.Render(chunk))
				fullResponse.WriteString(chunk)
				return nil
			},
		)
		
		if s.Active() {
			s.Stop()
		}
		
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			continue
		}
		
		// Render markdown
		markdown := fullResponse.String()
		renderedMd := renderer.RenderMarkdown(markdown)
		fmt.Print(renderedMd)
		fmt.Println("\n")
				// Extract and create files from response
		createdFiles := extractAndCreateFiles(markdown)
		if len(createdFiles) > 0 {
			fmt.Println("\033[1;32m✓ Created files:\033[0m")
			for _, file := range createdFiles {
				fmt.Printf("  - %s\n", file)
			}
			fmt.Println()
		}
				// Add assistant response to history
		sess.AddMessage("assistant", markdown)
		
		// Save session
		if err := sess.Save(); err != nil {
			fmt.Printf("Warning: failed to save session: %v\n", err)
		}
	}
	
	return nil
}
