package modes

import (
	"strings"

	"github.com/yourusername/llamasidekick/internal/session"
)

// BuildConversationContext formats session history into a single prompt.
// The last user message is substituted with enhancedLastUserMessage (typically including loaded file contents).
func BuildConversationContext(sess *session.Session, enhancedLastUserMessage string) string {
	var conversation strings.Builder

	for i, msg := range sess.History {
		switch msg.Role {
		case "user":
			conversation.WriteString("User: ")
			if i == len(sess.History)-1 {
				conversation.WriteString(enhancedLastUserMessage)
			} else {
				conversation.WriteString(msg.Content)
			}
			conversation.WriteString("\n\n")
		case "assistant":
			conversation.WriteString("Assistant: ")
			conversation.WriteString(msg.Content)
			conversation.WriteString("\n\n")
		}
	}

	return conversation.String()
}
