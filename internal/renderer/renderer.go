package renderer

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
)

var mdRenderer *glamour.TermRenderer

func init() {
	var err error
	// Create a dark-mode terminal renderer with specific style
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		// Print error to stderr for debugging
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize glamour renderer: %v\n", err)
		mdRenderer = nil
	}
}

// RenderMarkdown renders markdown text with glamour for terminal display
func RenderMarkdown(markdown string) string {
	if mdRenderer == nil {
		return markdown // Fallback to plain text
	}

	rendered, err := mdRenderer.Render(markdown)
	if err != nil {
		// Print error for debugging
		fmt.Fprintf(os.Stderr, "Warning: Failed to render markdown: %v\n", err)
		return markdown // Fallback on error
	}

	return strings.TrimSpace(rendered)
}

// StreamingMarkdownBuffer accumulates markdown chunks for rendering
type StreamingMarkdownBuffer struct {
	buffer strings.Builder
}

// NewStreamingMarkdownBuffer creates a new buffer
func NewStreamingMarkdownBuffer() *StreamingMarkdownBuffer {
	return &StreamingMarkdownBuffer{}
}

// Write adds a chunk to the buffer
func (b *StreamingMarkdownBuffer) Write(chunk string) {
	b.buffer.WriteString(chunk)
}

// String returns the accumulated content
func (b *StreamingMarkdownBuffer) String() string {
	return b.buffer.String()
}

// Render returns the glamour-rendered version
func (b *StreamingMarkdownBuffer) Render() string {
	return RenderMarkdown(b.buffer.String())
}
