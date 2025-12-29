package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client represents an Ollama API client
type Client struct {
	Host    string
	Model   string
	Debug   bool
	Version string
	client  *http.Client
}

// NewClient creates a new Ollama client
func NewClient(host, model string) *Client {
	return &Client{
		Host:   host,
		Model:  model,
		client: &http.Client{},
	}
}

// GenerateRequest represents a request to the Ollama generate API
type GenerateRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	System      string  `json:"system,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream"`
	Format      string  `json:"format,omitempty"`
}

// GenerateResponse represents a response from the Ollama generate API
type GenerateResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// StreamCallback is called for each chunk of the response
type StreamCallback func(chunk string) error

// GenerateJSON generates with JSON format constraint (non-streaming)
func (c *Client) GenerateJSON(model, prompt, system string, temperature float64) (string, error) {
	reqBody := GenerateRequest{
		Model:       model,
		Prompt:      prompt,
		System:      system,
		Temperature: temperature,
		Stream:      false,
		Format:      "json",
	}
	
	if c.Debug {
		fmt.Println("\n\033[38;5;240m=== DEBUG: JSON Request to Ollama ===")
		if c.Version != "" {
			fmt.Printf("LlamaSidekick Version: %s\n", c.Version)
		}
		fmt.Printf("Model: %s\n", reqBody.Model)
		fmt.Printf("Format: json\n")
		fmt.Printf("Temperature: %.2f\n", reqBody.Temperature)
		fmt.Printf("System Prompt: %s\n", system)
		fmt.Printf("User Prompt: %s\n", prompt)
		fmt.Println("=== END DEBUG ===")
		fmt.Println("\033[0m")
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	url := strings.TrimSuffix(c.Host, "/") + "/api/generate"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	
	if c.Debug {
		fmt.Println("\n\033[38;5;240m=== DEBUG: JSON Response from Ollama ===")
		fmt.Printf("Response: %s\n", result.Response)
		fmt.Println("=== END DEBUG ===")
		fmt.Println("\033[0m")
	}
	
	return result.Response, nil
}

// Generate sends a prompt to Ollama and streams the response
func (c *Client) Generate(prompt, system string, temperature float64, callback StreamCallback) error {
	reqBody := GenerateRequest{
		Model:       c.Model,
		Prompt:      prompt,
		System:      system,
		Temperature: temperature,
		Stream:      true,
	}
	
	if c.Debug {
		fmt.Println("\n\033[38;5;240m=== DEBUG: Request to Ollama ===")
		if c.Version != "" {
			fmt.Printf("LlamaSidekick Version: %s\n", c.Version)
		}
		fmt.Printf("Model: %s\n", reqBody.Model)
		fmt.Printf("Temperature: %.2f\n", reqBody.Temperature)
		fmt.Printf("System Prompt: %s\n", system)
		fmt.Printf("User Prompt: %s\n", prompt)
		fmt.Println("=== END DEBUG ===")
		fmt.Println("\033[0m")
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	url := strings.TrimSuffix(c.Host, "/") + "/api/generate"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama API error: %s - %s", resp.Status, string(body))
	}
	
	// Stream the response
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		var genResp GenerateResponse
		if err := json.Unmarshal([]byte(line), &genResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		
		if genResp.Response != "" {
			if err := callback(genResp.Response); err != nil {
				return err
			}
		}
		
		if genResp.Done {
			break
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	
	return nil
}

// Model represents an Ollama model
type Model struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

// ListModelsResponse represents the response from /api/tags
type ListModelsResponse struct {
	Models []Model `json:"models"`
}

// ListModels retrieves all available models from Ollama
func (c *Client) ListModels() ([]Model, error) {
	url := strings.TrimSuffix(c.Host, "/") + "/api/tags"
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %s", resp.Status)
	}
	
	var modelsResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}
	
	return modelsResp.Models, nil
}

// CheckConnection verifies that Ollama is running and accessible
func (c *Client) CheckConnection() error {
	_, err := c.ListModels()
	return err
}

// GenerateWithModel sends a prompt to Ollama using a specific model
func (c *Client) GenerateWithModel(model, prompt, system string, temperature float64, callback StreamCallback) error {
	reqBody := GenerateRequest{
		Model:       model,
		Prompt:      prompt,
		System:      system,
		Temperature: temperature,
		Stream:      true,
	}
	
	if c.Debug {
		fmt.Println("\n\033[38;5;240m=== DEBUG: Request to Ollama ===")
		if c.Version != "" {
			fmt.Printf("LlamaSidekick Version: %s\n", c.Version)
		}
		fmt.Printf("Model: %s\n", reqBody.Model)
		fmt.Printf("Temperature: %.2f\n", reqBody.Temperature)
		fmt.Printf("System Prompt: %s\n", system)
		fmt.Printf("User Prompt: %s\n", prompt)
		fmt.Println("=== END DEBUG ===")
		fmt.Println("\033[0m")
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	url := strings.TrimSuffix(c.Host, "/") + "/api/generate"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama API error: %s - %s", resp.Status, string(body))
	}
	
	// Stream the response
	scanner := bufio.NewScanner(resp.Body)
	var fullDebugResponse strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		var genResp GenerateResponse
		if err := json.Unmarshal([]byte(line), &genResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		
		if genResp.Response != "" {
			if c.Debug {
				fullDebugResponse.WriteString(genResp.Response)
			}
			if err := callback(genResp.Response); err != nil {
				return err
			}
		}
		
		if genResp.Done {
			if c.Debug {
				fmt.Println("\n\033[38;5;240m=== DEBUG: Response from Ollama ===")
				fmt.Printf("Full Response: %s\n", fullDebugResponse.String())
				fmt.Println("=== END DEBUG ===")
				fmt.Println("\033[0m")
			}
			break
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	
	return nil
}
