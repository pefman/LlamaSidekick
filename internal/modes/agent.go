package modes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// ProcessInput handles a single agent input with file creation support
func (m *AgentMode) ProcessInput(client *ollama.Client, sess *session.Session, cfg *config.Config, input string) error {
	modelName := cfg.GetModelForMode("agent")
	var responseText string
	
	// Detect if this is a file creation request
	lowerInput := strings.ToLower(input)
	needsFileCreation := strings.Contains(lowerInput, "create") && 
		(strings.Contains(lowerInput, "file") || 
		 strings.Contains(input, ".") || 
		 strings.Contains(lowerInput, "script") ||
		 strings.Contains(lowerInput, "html") ||
		 strings.Contains(lowerInput, "python") ||
		 strings.Contains(lowerInput, "javascript"))
	
	if client.Debug {
		fmt.Printf("\n[DEBUG] File creation detection: %v (input: %s)\n", needsFileCreation, input)
	}
	
	if needsFileCreation {
		// Use JSON mode for guaranteed file creation
		fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("blue")).Render("\nAgent: "))
		fmt.Println("Creating files...")
		
		jsonSystemPrompt := `You MUST respond with ONLY a valid JSON array of file objects. No markdown, no explanations, no extra text.

Each object must have exactly these fields:
- "filename": string (the file path/name)
- "content": string (the complete file content)

Example response format:
[{"filename": "test.txt", "content": "hello world"}]

For multiple files:
[{"filename": "index.html", "content": "<!DOCTYPE html>..."}, {"filename": "style.css", "content": "body {...}"}]

Output ONLY the JSON array. Any other text will cause failure.`
		
		jsonResponse, err := client.GenerateJSON(modelName, input, jsonSystemPrompt, 0.3)
		if err != nil {
			return fmt.Errorf("error generating JSON: %w", err)
		}
		
// Parse JSON response - handle both array and single object
	var files []struct {
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	
	// Try to unmarshal as array first
	if err := json.Unmarshal([]byte(jsonResponse), &files); err != nil {
		// If that fails, try as single object
		var singleFile struct {
			Filename string `json:"filename"`
			Content  string `json:"content"`
		}
		if err2 := json.Unmarshal([]byte(jsonResponse), &singleFile); err2 != nil {
			return fmt.Errorf("error parsing JSON response: %w\nResponse was: %s", err, jsonResponse)
		}
		// Wrap single file in array
		files = []struct {
			Filename string `json:"filename"`
			Content  string `json:"content"`
		}{singleFile}
		}
		
		if client.Debug {
			fmt.Printf("[DEBUG] Parsed %d files from JSON response\n", len(files))
		}
		
		// Create files
		for _, file := range files {
			// Create directory if needed
			dir := filepath.Dir(file.Filename)
			if dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					fmt.Printf("\033[38;5;9mError creating directory %s: %v\033[0m\n", dir, err)
					continue
				}
			}
			
			// Write file
			if err := os.WriteFile(file.Filename, []byte(file.Content), 0644); err != nil {
				fmt.Printf("\033[38;5;9mError creating file %s: %v\033[0m\n", file.Filename, err)
				continue
			}
			
			fmt.Printf("\033[1;32m✓ Created: %s\033[0m (%d bytes)\n", file.Filename, len(file.Content))
		}
		fmt.Println()
		
		responseText = fmt.Sprintf("Created %d file(s) successfully", len(files))
		
	} else {
		// Normal streaming response for non-file-creation tasks
		// Start spinner
		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Suffix = " Thinking..."
		s.Start()
		
		var fullResponse strings.Builder
		err := client.GenerateWithModel(
			modelName,
			input,
			m.GetSystemPrompt(),
			cfg.Ollama.Temperature,
			func(chunk string) error {
				if s.Active() {
					s.Stop()
					fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("blue")).Render("\nAgent: "))
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
		
		// Render markdown
		markdown := fullResponse.String()
		renderedMd := renderer.RenderMarkdown(markdown)
		fmt.Print(renderedMd)
		fmt.Println("\n")
		
		responseText = markdown
	}
	
	// Add assistant response to history
	sess.AddMessage("assistant", responseText)
	
	// Save session
	if err := sess.Save(); err != nil {
		fmt.Printf("Warning: failed to save session: %v\n", err)
	}
	
	return nil
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
		
		// Process the input (handles file creation and normal responses)
		if err := m.ProcessInput(client, sess, cfg, input); err != nil {
			fmt.Printf("\nError: %v\n", err)
		}
	}
	
	return nil
}
