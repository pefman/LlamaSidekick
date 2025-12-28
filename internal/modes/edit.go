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

// EditMode helps with code editing and modifications
type EditMode struct{}

func (m *EditMode) Name() string {
	return "Edit"
}

func (m *EditMode) Description() string {
	return "Get help editing code with suggestions and diffs - automatically reads referenced files"
}

func (m *EditMode) GetSystemPrompt() string {
	return `You are an expert code editor assistant. Your role is to help developers edit and improve their code.

When helping with edits:
1. Understand the context and intent of the change
2. Suggest specific, actionable modifications
3. Explain why the changes improve the code
4. Consider edge cases and potential issues
5. Provide diffs or clear before/after examples when helpful

The user's message may include file contents automatically loaded from their working directory.
When you see "File contents:" followed by file content, use that context to provide specific suggestions.

Always prioritize code quality, readability, and best practices.

FORMATTING:
- Use markdown for clear formatting
- Code blocks with triple backticks and language syntax
- Use bold (**text**) for emphasis
- Use headers (##) to organize sections
- Keep explanations clear and concise`
}

func (m *EditMode) Run(client *ollama.Client, sess *session.Session, cfg *config.Config) error {
	sess.SetMode(ModeEdit)
	
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("green")).Render("\n=== EDIT MODE ==="))
	fmt.Println("Get help editing code and making modifications.")
	fmt.Println("Type 'exit' to return to main menu.\n")
	
	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Render("edit> "))
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
		
		// Detect and read files mentioned in the input
		enhancedInput := ReadFilesFromInput(input)
		
		// Add user message to history
		sess.AddMessage("user", input)
		
		// Start spinner
		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Suffix = " Thinking..."
		s.Start()
		
		var fullResponse strings.Builder
		modelName := cfg.GetModelForMode("edit")
		err = client.GenerateWithModel(
			modelName,
			enhancedInput,
			m.GetSystemPrompt(),
			cfg.Ollama.Temperature,
			func(chunk string) error {
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
		sess.AddMessage("assistant", fullResponse.String())
		
		// Save session
		if err := sess.Save(); err != nil {
			fmt.Printf("Warning: failed to save session: %v\n", err)
		}
	}
	
	return nil
}
