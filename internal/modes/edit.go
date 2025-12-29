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
	"github.com/yourusername/llamasidekick/internal/safeio"
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
	return "You are an expert code editor assistant. Your role is to help developers edit and improve their code.\n\n" +
		"When helping with edits:\n" +
		"1. Understand the context and intent of the change\n" +
		"2. Suggest specific, actionable modifications\n" +
		"3. Explain why the changes improve the code\n" +
		"4. Consider edge cases and potential issues\n" +
		"5. Provide diffs or clear before/after examples when helpful\n\n" +
		"The user's message may include file contents automatically loaded from their working directory.\n" +
		"When you see \"File contents:\" followed by file content, use that context to provide specific suggestions.\n\n" +
		"Always prioritize code quality, readability, and best practices.\n\n" +
		"FORMATTING:\n" +
		"- Use markdown for clear formatting\n" +
		"- Code blocks with triple backticks and language syntax\n" +
		"- Use bold (**text**) for emphasis\n" +
		"- Use headers (##) to organize sections\n" +
		"- Keep explanations clear and concise"
}

// ProcessInput handles a single edit request with automatic file modification
func (m *EditMode) ProcessInput(client *ollama.Client, sess *session.Session, cfg *config.Config, input string) error {
	sess.SetMode(ModeEdit)
	enhancedInput := ReadFilesFromInputWithRoot(input, sess.ProjectRoot)
	sess.AddMessage("user", input)

	fileToEdit := detectFileInInput(input)
	if fileToEdit == "" {
		fileToEdit = sess.LastEditedFile
	} else {
		// Record the explicit filename the user referenced.
		sess.SetLastEditedFile(fileToEdit)
	}
	
	if fileToEdit != "" {
		absPath, relPath, err := safeio.ResolveWithinRoot(sess.ProjectRoot, fileToEdit)
		if err != nil {
			return fmt.Errorf("refusing to edit '%s': %w", fileToEdit, err)
		}
		fileToEdit = relPath
		if !fileExists(absPath) {
			// Fall back to suggestion mode if the resolved file doesn't exist.
			goto suggestionMode
		}
		// File editing mode
		currentContent, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", relPath, err)
		}

		if client.Debug {
			fmt.Printf("\n[DEBUG] File editing detected: %s (%d bytes)\n", relPath, len(currentContent))
		}

		fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Render("\nEdit: "))
		fmt.Printf("Modifying %s...\n", relPath)

		jsonSystemPrompt := "You MUST respond with ONLY a valid JSON object. No markdown, no explanations, no extra text.\n\n" +
			"The object must have exactly these fields:\n" +
			"- filename: string (the file path/name being edited)\n" +
			"- content: string (the COMPLETE modified file content)\n" +
			"- summary: string (brief description of changes made)\n\n" +
			"Example response format:\n" +
			"{\"filename\": \"index.html\", \"content\": \"full content here\", \"summary\": \"Reduced animation speed\"}\n\n" +
			"Output ONLY the JSON object. Any other text will cause failure."

		conversationContext := BuildConversationContext(sess, enhancedInput)
		editPrompt := fmt.Sprintf("File: %s\n\nCurrent content:\n%s\n\nUser request: %s\n\nProvide the COMPLETE modified file content.",
			relPath, string(currentContent), input)
		fullPrompt := conversationContext + "\n\n" + editPrompt

		modelName := cfg.GetModelForMode("edit")
		jsonResponse, err := client.GenerateJSON(modelName, fullPrompt, jsonSystemPrompt, 0.3)
		if err != nil {
			return fmt.Errorf("error generating JSON: %w", err)
		}

		type EditResult struct {
			Filename string `json:"filename"`
			Content  string `json:"content"`
			Summary  string `json:"summary"`
		}
		var result EditResult

		if err := json.Unmarshal([]byte(jsonResponse), &result); err != nil {
			return fmt.Errorf("error parsing JSON response: %w\nResponse was: %s", err, jsonResponse)
		}

		if client.Debug {
			fmt.Printf("[DEBUG] Parsed edit result: %s - %s\n", result.Filename, result.Summary)
		}

		backupPath, err := safeio.WriteFileWithBackup(absPath, []byte(result.Content))
		if err != nil {
			return fmt.Errorf("error writing file: %w", err)
		}

		fmt.Printf("\033[1;32m✓ Modified: %s\033[0m (%d → %d bytes)\n", relPath, len(currentContent), len(result.Content))
		fmt.Printf("  %s\n", result.Summary)
		if backupPath != "" {
			fmt.Printf("\033[38;5;240m  Backup saved: %s\033[0m\n\n", backupPath)
		} else {
			fmt.Println()
		}

		sess.SetLastEditedFile(relPath)
		responseText := fmt.Sprintf("Modified %s: %s", relPath, result.Summary)
		sess.AddMessage("assistant", responseText)

		if err := sess.Save(); err != nil {
			fmt.Printf("Warning: failed to save session: %v\n", err)
		}

		return nil
	}

suggestionMode:
	// Suggestion mode (no file editing)
		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Suffix = " Thinking..."
		s.Start()
		
		var fullResponse strings.Builder
		modelName := cfg.GetModelForMode("edit")
		conversationContext := BuildConversationContext(sess, enhancedInput)
		err := client.GenerateWithModel(
			modelName,
			conversationContext,
			m.GetSystemPrompt(),
			cfg.Ollama.Temperature,
			func(chunk string) error {
				if s.Active() {
					s.Stop()
					fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Render("\nEdit: "))
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
		
		sess.AddMessage("assistant", fullResponse.String())
	if err := sess.Save(); err != nil {
		fmt.Printf("Warning: failed to save session: %v\n", err)
	}
	return nil
}

func (m *EditMode) Run(client *ollama.Client, sess *session.Session, cfg *config.Config) error {
	sess.SetMode(ModeEdit)
	
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("green")).Render("\n=== EDIT MODE ==="))
	fmt.Println("Get help editing code and making modifications.")
	fmt.Println("Type 'exit' to return to main menu.")
	fmt.Println()
	
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
		
		if err := m.ProcessInput(client, sess, cfg, input); err != nil {
			fmt.Printf("\nError: %v\n", err)
		}
	}
	
	return nil
}

func detectFileInInput(input string) string {
	extensions := []string{".html", ".js", ".css", ".go", ".py", ".java", ".cpp", ".c", ".h", 
		".txt", ".json", ".xml", ".yml", ".yaml", ".md", ".ts", ".tsx", ".jsx", 
		".php", ".rb", ".rs", ".sh", ".bat"}
	
	words := strings.Fields(input)
	for _, word := range words {
		for _, ext := range extensions {
			if strings.HasSuffix(word, ext) {
				return filepath.Clean(word)
			}
		}
	}
	return ""
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
