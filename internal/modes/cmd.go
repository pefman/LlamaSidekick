package modes

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/ollama"
	"github.com/yourusername/llamasidekick/internal/session"
)

var cmdStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("yellow")).
	Bold(true)

var copiedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("green")).
	Bold(true)

// CmdMode helps generate commands without executing them
type CmdMode struct{}

func (m *CmdMode) Name() string {
	return "CMD"
}

func (m *CmdMode) Description() string {
	return "Get help with commands - generates but never executes"
}

func (m *CmdMode) GetSystemPrompt() string {
	osType := "Linux/Unix"
	shellType := "bash"
	exampleCmd := "df -h"
	
	if runtime.GOOS == "windows" {
		osType = "Windows"
		shellType = "PowerShell"
		exampleCmd = "Get-PSDrive -PSProvider FileSystem | Select-Object Name, Used, Free"
	}
	
	return fmt.Sprintf("You are a command-line expert assistant. Generate ONLY the exact command to run.\n\n"+
		"USER'S OPERATING SYSTEM: %s\n"+
		"SHELL: %s\n\n"+
		"CRITICAL OUTPUT FORMAT:\n"+
		"- Output ONLY the command itself for %s\n"+
		"- NO markdown formatting\n"+
		"- NO code blocks\n"+
		"- NO backticks\n"+
		"- NO explanations or descriptions\n"+
		"- JUST the raw command ready to paste into a %s terminal\n\n"+
		"Example user: \"check disk space\"\n"+
		"CORRECT output: %s\n"+
		"WRONG output: Here's how... ```bash df -h```\n\n"+
		"Output the command only.", osType, shellType, osType, osType, exampleCmd)
}

// ProcessInput handles a single cmd request.
func (m *CmdMode) ProcessInput(client *ollama.Client, sess *session.Session, cfg *config.Config, input string) error {
	sess.SetMode(ModeCmd)
	modelName := cfg.GetModelForMode("cmd")

	enhancedInput := ReadFilesFromInputWithRoot(input, sess.ProjectRoot)
	sess.AddMessage("user", input)

	conversationContext := BuildConversationContext(sess, enhancedInput)

	// Start spinner
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Generating command..."
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
				fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("yellow")).Render("\nCommands:\n"))
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
		return fmt.Errorf("error generating response: %w", err)
	}

	fmt.Println()

	response := fullResponse.String()
	commands := extractCommands(response)
	if len(commands) > 0 {
		cmdToCopy := strings.Join(commands, "\n")
		if err := clipboard.WriteAll(cmdToCopy); err != nil {
			fmt.Printf("Warning: failed to copy to clipboard: %v\n", err)
		} else {
			fmt.Println(copiedStyle.Render("✓ Command(s) copied to clipboard - ready to paste!"))
		}
	}

	fmt.Println()

	sess.AddMessage("assistant", response)
	if err := sess.Save(); err != nil {
		fmt.Printf("Warning: failed to save session: %v\n", err)
	}

	return nil
}

func (m *CmdMode) Run(client *ollama.Client, sess *session.Session, cfg *config.Config) error {
	sess.SetMode(ModeCmd)
	
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("yellow")).Render("\n=== CMD MODE ==="))
	fmt.Println("Get command help - commands are copied to clipboard, NEVER executed.")
	fmt.Println("Type 'exit' to return to main menu.")
	fmt.Println()
	
	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Render("cmd> "))
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

// extractCommands extracts commands from code blocks in the response
func extractCommands(response string) []string {
	// Match code blocks with ```bash, ```powershell, ```sh, or just ```
	re := regexp.MustCompile("```(?:bash|powershell|sh|shell)?\n([^`]+)```")
	matches := re.FindAllStringSubmatch(response, -1)
	
	var commands []string
	for _, match := range matches {
		if len(match) > 1 {
			cmd := strings.TrimSpace(match[1])
			if cmd != "" {
				commands = append(commands, cmd)
			}
		}
	}
	
	return commands
}
