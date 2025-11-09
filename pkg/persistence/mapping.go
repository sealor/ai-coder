package persistence

import (
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

func NewSessionFromOpenAI(param *openai.ChatCompletionNewParams) *Session {
	var session Session
	session.Model = param.Model
	session.Reasoning = string(param.ReasoningEffort)

	for _, paramMessage := range param.Messages {
		sessionMessage := Message{Role: *paramMessage.GetRole()}

		if m := paramMessage.OfAssistant; m != nil {
			sessionMessage = Message{Role: "assistant", Content: m.Content.OfString.String(), ToolCalls: NewToolCallsFromOpenAI(m.ToolCalls)}
		} else if m := paramMessage.OfDeveloper; m != nil {
			sessionMessage = Message{Role: "developer", Content: m.Content.OfString.String()}
		} else if m := paramMessage.OfSystem; m != nil {
			sessionMessage = Message{Role: "system", Content: m.Content.OfString.String()}
		} else if m := paramMessage.OfTool; m != nil {
			sessionMessage = Message{Role: "tool", Content: m.Content.OfString.String(), ToolCallID: m.ToolCallID}
		} else if m := paramMessage.OfUser; m != nil {
			sessionMessage = Message{Role: "user", Content: m.Content.OfString.String()}
		}

		session.Messages = append(session.Messages, sessionMessage)
	}

	return &session
}

func NewToolCallsFromOpenAI(calls []openai.ChatCompletionMessageToolCallUnionParam) []ToolCall {
	var toolCalls []ToolCall
	for _, call := range calls {
		toolCalls = append(toolCalls, *NewToolCallFromOpenAI(&call))
	}
	return toolCalls
}

func NewToolCallFromOpenAI(call *openai.ChatCompletionMessageToolCallUnionParam) *ToolCall {
	if call.OfFunction != nil {
		f := call.OfFunction
		return &ToolCall{f.ID, f.Function.Name, f.Function.Arguments}
	}

	return &ToolCall{ID: fmt.Sprint("ERROR: mapping failed for", call)}
}

func NewParamsFromSession(session *Session) *openai.ChatCompletionNewParams {
	var params openai.ChatCompletionNewParams

	params.Model = session.Model
	params.ReasoningEffort = shared.ReasoningEffort(session.Reasoning)

	for _, sessionMessage := range session.Messages {
		var paramMessage openai.ChatCompletionMessageParamUnion

		switch sessionMessage.Role {
		case "assistant":
			paramMessage = openai.AssistantMessage(sessionMessage.Content)
			paramMessage.OfAssistant.ToolCalls = NewToolCallsFromSession(sessionMessage.ToolCalls)
		case "developer":
			paramMessage = openai.DeveloperMessage(sessionMessage.Content)
		case "system":
			paramMessage = openai.SystemMessage(sessionMessage.Content)
		case "tool":
			paramMessage = openai.ToolMessage(sessionMessage.Content, sessionMessage.ToolCallID)
		case "user":
			paramMessage = openai.UserMessage(sessionMessage.Content)
		}

		params.Messages = append(params.Messages, paramMessage)
	}

	return &params
}

func NewToolCallsFromSession(calls []ToolCall) []openai.ChatCompletionMessageToolCallUnionParam {
	var toolCalls []openai.ChatCompletionMessageToolCallUnionParam
	for _, call := range calls {
		toolCalls = append(toolCalls, *NewToolCallFromSession(&call))
	}
	return toolCalls
}

func NewToolCallFromSession(call *ToolCall) *openai.ChatCompletionMessageToolCallUnionParam {
	return &openai.ChatCompletionMessageToolCallUnionParam{
		OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
			ID:       call.ID,
			Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{Name: call.Name, Arguments: call.Arguments},
		},
	}
}
