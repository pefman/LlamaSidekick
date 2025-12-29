package modes

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
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

The user's message may include file contents automatically loaded from their working directory.
When you see "File contents:" followed by file content, analyze and explain that specific content.

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

// ProcessInput handles a single ask request.
func (m *AskMode) ProcessInput(client *ollama.Client, sess *session.Session, cfg *config.Config, input string) error {
	sess.SetMode(ModeAsk)
	modelName := cfg.GetModelForMode("ask")

	// Detect and read files mentioned in the input
	enhancedInput := ReadFilesFromInputWithRoot(input, sess.ProjectRoot)

	// Add user message to history
	sess.AddMessage("user", input)

	// Build conversation context from session history
	conversationContext := BuildConversationContext(sess, enhancedInput)

	// Start spinner
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Thinking..."
	s.Start()

	var fullResponse strings.Builder
	err := client.GenerateWithModel(
		modelName,
		conversationContext,
		m.GetSystemPrompt(),
		cfg.Ollama.Temperature,
		func(chunk string) error {
			if s.Active() {
				s.Stop()
				fmt.Println()
			}
			fullResponse.WriteString(chunk)
			return nil
		},
	)

	if s.Active() {
		s.Stop()
	}

	if err != nil {
		return err
	}

	response := fullResponse.String()

	// Render the markdown response
	rendered := renderer.RenderMarkdown(response)
	fmt.Println(rendered)

	sess.AddMessage("assistant", response)
	if err := sess.Save(); err != nil {
		fmt.Printf("Warning: failed to save session: %v\n", err)
	}

	return nil
}

func (m *AskMode) Run(client *ollama.Client, sess *session.Session, cfg *config.Config) error {
	fmt.Println("\n\033[1;38;5;75m=== Ask Mode ===\033[0m")
	fmt.Println("\033[38;5;240mGet answers and information without any changes or plans\033[0m")
	fmt.Println("\033[38;5;240mType 'q' to return to menu\033[0m")
	fmt.Println()

	sess.SetMode(ModeAsk)
	reader := bufio.NewReader(os.Stdin)

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

		if err := m.ProcessInput(client, sess, cfg, input); err != nil {
			fmt.Printf("\n\033[38;5;9mError: %v\033[0m\n", err)
			continue
		}
	}
}
