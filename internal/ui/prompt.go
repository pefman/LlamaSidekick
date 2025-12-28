package ui

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/briandowns/spinner"
	"github.com/chzyer/readline"
	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/modes"
	"github.com/yourusername/llamasidekick/internal/ollama"
	"github.com/yourusername/llamasidekick/internal/renderer"
	"github.com/yourusername/llamasidekick/internal/session"
)

// autoCompleter provides tab completion for commands
type autoCompleter struct{}

func (a *autoCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	lineStr := string(line)
	
	// Only autocomplete at the beginning of the line
	if !strings.HasPrefix(lineStr, "/") {
		return nil, 0
	}
	
	commands := []string{"/plan", "/edit", "/agent", "/cmd", "/ask", "/menu", "/clear"}
	
	var suggestions [][]rune
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, lineStr) {
			suggestions = append(suggestions, []rune(cmd[len(lineStr):]))
		}
	}
	
	return suggestions, len(lineStr)
}

// RunPrompt shows a command prompt that accepts /mode commands or 'm' for menu
func RunPrompt(cfg *config.Config, client *ollama.Client, sess *session.Session) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     "/tmp/llamasidekick_history",
		AutoComplete:    &autoCompleter{},
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()
	
	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		
		input := strings.TrimSpace(line)
		
		if input == "" {
			continue
		}
		
		// Check for quit
		if input == "q" || input == "quit" || input == "exit" {
			return nil
		}
		
		// Check for menu (support both 'm' and 'menu')
		if input == "m" || input == "menu" {
			// Show menu and wait for selection
			if err := ShowMenu(cfg, client, sess); err != nil {
				return err
			}
			continue
		}
		
		// Check for clear command
		if input == "/clear" || input == "clear" {
			// Clear the conversation history
			sess.History = []session.Message{}
			if err := sess.Save(); err != nil {
				fmt.Printf("\033[38;5;9mError saving session: %v\033[0m\n", err)
			} else {
				fmt.Println("\033[38;5;10mConversation history cleared!\033[0m")
			}
			continue
		}
		
		// Parse slash commands
		if strings.HasPrefix(input, "/") {
			parts := strings.SplitN(input, " ", 2)
			command := strings.TrimPrefix(parts[0], "/")
			prompt := ""
			if len(parts) > 1 {
				prompt = parts[1]
			}
			
			var mode modes.Mode
			switch command {
			case "plan":
				mode = &modes.PlanMode{}
			case "edit":
				mode = &modes.EditMode{}
			case "agent":
				mode = &modes.AgentMode{}
			case "cmd":
				mode = &modes.CmdMode{}
			case "ask":
				mode = &modes.AskMode{}
			default:
				fmt.Printf("\033[38;5;9mUnknown command: /%s\033[0m\n", command)
				fmt.Println("\033[38;5;240mAvailable commands: /plan, /edit, /agent, /cmd, /ask, /clear, or 'm' for menu\033[0m")
				continue
			}
			
			// Save debug snapshot before clearing if debug mode is enabled
			if cfg.Ollama.Debug && len(sess.History) > 0 {
				if err := sess.SaveDebug(command); err != nil {
					fmt.Printf("\033[38;5;9mError saving debug session: %v\033[0m\n", err)
				}
			}
			
			// Clear session history for fresh start
			sess.History = []session.Message{}
			sess.Save()
			
			// If there's a prompt, execute directly
			if prompt != "" {
				if err := executeQuickCommand(mode, client, sess, cfg, prompt); err != nil {
					fmt.Printf("\033[38;5;9mError: %v\033[0m\n", err)
				}
			} else {
				// No prompt, enter interactive mode
				if err := mode.Run(client, sess, cfg); err != nil {
					return err
				}
			}
			continue
		}
		
		// Default: treat as a quick /plan command
		mode := &modes.PlanMode{}
		if err := executeQuickCommand(mode, client, sess, cfg, input); err != nil {
			fmt.Printf("\033[38;5;9mError: %v\033[0m\n", err)
		}
	}
	
	return nil
}

// executeQuickCommand executes a single command and returns to prompt
func executeQuickCommand(mode modes.Mode, client *ollama.Client, sess *session.Session, cfg *config.Config, prompt string) error {
	// Detect and read files from the prompt
	enhancedPrompt := modes.ReadFilesFromInput(prompt)
	
	sess.AddMessage("user", prompt)
	
	fmt.Print("\n\033[1;38;5;170m" + mode.Name() + ":\033[0m ")
	
	var fullResponse strings.Builder
	var modeStr string
	switch mode.(type) {
	case *modes.PlanMode:
		modeStr = "plan"
	case *modes.EditMode:
		modeStr = "edit"
	case *modes.AgentMode:
		modeStr = "agent"
	case *modes.CmdMode:
		modeStr = "cmd"
	case *modes.AskMode:
		modeStr = "ask"
	}
	
	modelName := cfg.GetModelForMode(modeStr)
	
	// Print mode header for CMD mode
	if modeStr == "cmd" {
		fmt.Print("\n\033[1;33mCMD:\033[0m ")
	}
	
	// Build conversation context from session history
	var conversationContext strings.Builder
	for i, msg := range sess.History {
		if msg.Role == "user" {
			conversationContext.WriteString("User: ")
			// Use enhanced prompt for the last user message
			if i == len(sess.History)-1 {
				conversationContext.WriteString(enhancedPrompt)
			} else {
				conversationContext.WriteString(msg.Content)
			}
			conversationContext.WriteString("\n\n")
		} else if msg.Role == "assistant" {
			conversationContext.WriteString("Assistant: ")
			conversationContext.WriteString(msg.Content)
			conversationContext.WriteString("\n\n")
		}
	}
	
	// Start spinner
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Thinking..."
	s.Start()
	
	err := client.GenerateWithModel(
		modelName,
		conversationContext.String(),
		mode.GetSystemPrompt(),
		cfg.Ollama.Temperature,
		func(chunk string) error {
			if s.Active() {
				s.Stop()
				fmt.Println() // Add newline after spinner
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
	
	// Render markdown for non-CMD modes
	if modeStr != "cmd" {
		renderedMd := renderer.RenderMarkdown(response)
		fmt.Println(renderedMd)
	} else {
		// CMD mode: just print plain text
		fmt.Println(response)
	}
	
	// Handle CMD mode clipboard copying
	if modeStr == "cmd" {
		// Copy the raw response (clean command) to clipboard
		cleanResponse := strings.TrimSpace(response)
		if cleanResponse != "" {
			if err := clipboard.WriteAll(cleanResponse); err == nil {
				fmt.Println("\n\033[1;32m✓ Copied to clipboard\033[0m")
			}
		}
	}
	
	fmt.Println("\n")
	
	sess.AddMessage("assistant", response)
	
	// Save session
	if err := sess.Save(); err != nil {
		fmt.Printf("\033[38;5;240mWarning: failed to save session: %v\033[0m\n", err)
	}
	
	return nil
}

// extractCommandsFromResponse extracts commands from code blocks
func extractCommandsFromResponse(response string) []string {
	re := regexp.MustCompile("```(?:bash|powershell|sh|shell)?\\n([^`]+)```")
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
