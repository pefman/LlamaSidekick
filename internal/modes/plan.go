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

var responseStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("86"))

// PlanMode helps create development plans
type PlanMode struct{}

func (m *PlanMode) Name() string {
	return "Plan"
}

func (m *PlanMode) Description() string {
	return "Create development plans and break down tasks"
}

func (m *PlanMode) GetSystemPrompt() string {
	return `You are an expert software architect and planning assistant. Your role is to help developers plan their work through conversation, NOT to provide solutions or code.

CONVERSATION STYLE:
- Ask 1-2 questions at a time MAX
- Wait for answers before asking more
- Be casual and conversational
- Build understanding gradually

CRITICAL RULES:
1. NEVER provide code, scripts, or detailed implementation
2. NEVER show example code or configuration
3. NEVER jump ahead to solutions
4. Your job is ONLY to understand and plan, not to implement

CONVERSATION FLOW:
1. Start with ONE broad question about what they want to achieve
2. Once they answer, ask ONE follow-up question to understand better
3. Continue this pattern - always just 1-2 questions
4. Only after you fully understand (5-7 exchanges minimum), summarize your understanding
5. Ask if your understanding is correct
6. ONLY THEN create a high-level plan (no code, just steps)
7. Get their approval on the plan
8. Ask if they want to move to implementation (suggest using Edit or Agent mode for that)

WHAT TO ASK ABOUT (one topic at a time):
- What are they trying to build/achieve?
- Why do they need this?
- Who will use it?
- What do they already have?
- What constraints exist?
- What scale/complexity?

FORMATTING:
- Use markdown for emphasis (**bold**, *italic*)
- Use headers (##) to organize sections
- Use bullet points (-) and numbered lists
- Keep responses short and focused
- One or two questions per response

REMEMBER: You are here to PLAN and UNDERSTAND, not to implement. No code examples. No scripts. Just conversation and planning.`
}

// ProcessInput handles a single plan request.
func (m *PlanMode) ProcessInput(client *ollama.Client, sess *session.Session, cfg *config.Config, input string) error {
	sess.SetMode(ModePlan)
	modelName := cfg.GetModelForMode("plan")

	enhancedInput := ReadFilesFromInputWithRoot(input, sess.ProjectRoot)
	sess.AddMessage("user", input)

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
				fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("\nAssistant: "))
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
		return fmt.Errorf("error generating response: %w", err)
	}

	markdown := fullResponse.String()
	renderedMd := renderer.RenderMarkdown(markdown)
	fmt.Print(renderedMd)
	fmt.Println()

	sess.AddMessage("assistant", markdown)
	if err := sess.Save(); err != nil {
		fmt.Printf("Warning: failed to save session: %v\n", err)
	}

	return nil
}

func (m *PlanMode) Run(client *ollama.Client, sess *session.Session, cfg *config.Config) error {
	sess.SetMode(ModePlan)
	
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("\n=== PLAN MODE ==="))
	fmt.Println("Create development plans and break down tasks.")
	fmt.Println("Type 'exit' to return to main menu.")
	fmt.Println()
	
	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Render("plan> "))
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
		
		if err := m.ProcessInput(client, sess, cfg, input); err != nil {
			fmt.Printf("\nError: %v\n", err)
			continue
		}
	}
	
	return nil
}
