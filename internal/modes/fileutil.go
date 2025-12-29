package modes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ReadFilesFromInput detects file references in input and reads their contents
func ReadFilesFromInput(input string) string {
	filePattern := regexp.MustCompile(`(?:^|\s)([a-zA-Z0-9_\-./\\]+\.(go|js|ts|py|java|c|cpp|h|rs|rb|php|cs|swift|kt|sh|bash|yml|yaml|json|xml|md|txt))(?:\s|$)`)
	matches := filePattern.FindAllStringSubmatch(input, -1)
	
	if len(matches) == 0 {
		return input
	}
	
	var fileContents strings.Builder
	fileContents.WriteString("\n\nFile contents:\n")
	
	for _, match := range matches {
		filename := match[1]
		
		// Try to read the file from current directory
		content, err := os.ReadFile(filename)
		if err != nil {
			// Try with absolute path
			absPath, _ := filepath.Abs(filename)
			content, err = os.ReadFile(absPath)
			if err != nil {
				fmt.Printf("\033[38;5;240m(Note: Could not read file '%s')\033[0m\n", filename)
				continue
			}
		}
		
		fileContents.WriteString(fmt.Sprintf("\n--- %s ---\n", filename))
		fileContents.WriteString(string(content))
		fileContents.WriteString(fmt.Sprintf("\n--- End of %s ---\n", filename))
	}
	
	if fileContents.Len() > len("\n\nFile contents:\n") {
		return input + fileContents.String()
	}
	
	return input
}

// extractAndCreateFiles finds code blocks with FILENAME: prefix and creates the files
func extractAndCreateFiles(response string) []string {
	var createdFiles []string
	
	// Pattern: FILENAME: path/to/file.ext followed by code block
	pattern := regexp.MustCompile(`(?i)FILENAME:\s*([^\n]+)\n\s*\x60\x60\x60[^\n]*\n([\s\S]*?)\x60\x60\x60`)
	matches := pattern.FindAllStringSubmatch(response, -1)
	
	fmt.Printf("\n[DEBUG] Checking for files to create... Found %d matches\n", len(matches))
	
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		
		filename := strings.TrimSpace(match[1])
		content := match[2]
		
		fmt.Printf("[DEBUG] Creating file: %s (%d bytes)\n", filename, len(content))
		
		// Create directory if needed
		dir := filepath.Dir(filename)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("\033[38;5;9mError creating directory %s: %v\033[0m\n", dir, err)
				continue
			}
		}
		
		// Write file
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			fmt.Printf("\033[38;5;9mError creating file %s: %v\033[0m\n", filename, err)
			continue
		}
		
		createdFiles = append(createdFiles, filename)
	}
	
	return createdFiles
}
