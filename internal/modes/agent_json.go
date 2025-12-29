package modes

import (
	"encoding/json"
	"fmt"
)

type GeneratedFile struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// ParseGeneratedFilesJSON parses either a JSON array of files or a single file object.
func ParseGeneratedFilesJSON(jsonResponse string) ([]GeneratedFile, error) {
	var files []GeneratedFile
	if err := json.Unmarshal([]byte(jsonResponse), &files); err == nil {
		return files, nil
	}

	var single GeneratedFile
	if err := json.Unmarshal([]byte(jsonResponse), &single); err != nil {
		return nil, fmt.Errorf("invalid JSON for generated files")
	}
	return []GeneratedFile{single}, nil
}
