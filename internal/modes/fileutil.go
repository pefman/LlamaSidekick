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
