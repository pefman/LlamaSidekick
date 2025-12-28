package modes

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/ollama"
	"github.com/yourusername/llamasidekick/internal/renderer"
	"github.com/yourusername/llamasidekick/internal/session"
)

// AskMode provides information and answers questions without making changes
type AskMode struct{}

func (m *AskMode) Name() string {
	return "Ask"
}

func (m *AskMode) Description() string {
	return "Get information and answers without any changes"
}

func (m *AskMode) GetSystemPrompt() string {
	return `You are a helpful information assistant. Your role is to provide clear, accurate information and answer questions.

CRITICAL RULES:
1. NEVER suggest making changes, edits, or implementations
2. NEVER provide plans or action items
3. NEVER offer to help with tasks - only provide information
4. Focus solely on answering questions and explaining concepts
5. Be concise and factual

YOUR RESPONSES SHOULD:
- Answer the question directly
- Explain concepts clearly
- Provide factual information
- Include examples only for clarity, never for implementation
- Stay neutral and informative

YOU MUST NOT:
- Suggest creating, editing, or modifying anything
- Provide step-by-step instructions for tasks
- Offer to help plan or implement solutions
- Give actionable advice beyond pure information

If asked how to do something, explain what it is and how it works conceptually, but don't provide implementation steps.`
}

func (m *AskMode) Run(client *ollama.Client, sess *session.Session, cfg *config.Config) error {
	fmt.Println("\n\033[1;38;5;75m=== Ask Mode ===\033[0m")
	fmt.Println("\033[38;5;240mGet answers and information without any changes or plans\033[0m")
	fmt.Println("\033[38;5;240mType 'q' to return to menu\033[0m\n")

	reader := bufio.NewReader(os.Stdin)
	modelName := cfg.GetModelForMode("ask")

	for {
		fmt.Print("\n\033[1;38;5;75mask>\033[0m ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if input == "q" || input == "quit" {
			return nil
		}

		sess.AddMessage("user", input)

		fmt.Print("\n\033[1;38;5;75mAsk:\033[0m ")

		var fullResponse strings.Builder
		err = client.GenerateWithModel(
			modelName,
			input,
			m.GetSystemPrompt(),
			cfg.Ollama.Temperature,
			func(chunk string) error {
				fullResponse.WriteString(chunk)
				return nil
			},
		)

		if err != nil {
			fmt.Printf("\n\033[38;5;9mError: %v\033[0m\n", err)
			continue
		}

		response := fullResponse.String()
		
		// Render the markdown response
		rendered := renderer.RenderMarkdown(response)
		fmt.Println(rendered)

		sess.AddMessage("assistant", response)
		sess.Save()
	}
}
