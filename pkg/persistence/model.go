// Package persistence handles mapping and YAML serialization of sessions
package persistence

type Session struct {
	Model     string
	Reasoning string

	Messages []Message
}

type Message struct {
	Role       string     `yaml:"role"`
	Content    string     `yaml:"content,omitempty"`
	ToolCallID string     `yaml:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `yaml:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Arguments string `yaml:"arguments"`
}
