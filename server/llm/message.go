package llm

// Message is a generic chat message for the multi-turn tool-calling loop.
// It is separate from the single-shot Generate path (which still uses
// a simple system+user pair) so stage 1-3 calls aren't forced through
// the heavier multi-turn type.
type Message struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCallReq `json:"tool_calls,omitempty"`
}

// NewUserMessage creates a user-role message.
func NewUserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// NewAssistantMessage creates an assistant-role message with plain text.
func NewAssistantMessage(content string) Message {
	return Message{Role: "assistant", Content: content}
}

// NewToolResultMessage creates a tool-result message (role "tool").
func NewToolResultMessage(toolCallID, content string) Message {
	return Message{Role: "tool", ToolCallID: toolCallID, Content: content}
}

// AssistantMessageWithToolCalls creates an assistant message that requested tool calls.
func AssistantMessageWithToolCalls(calls []ToolCallReq) Message {
	return Message{Role: "assistant", ToolCalls: calls}
}
