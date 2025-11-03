// Package tooling provides functions for the tool API
package tooling

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"
)

var ReadFileTool openai.ChatCompletionToolUnionParam = openai.ChatCompletionToolUnionParam{
	OfFunction: &openai.ChatCompletionFunctionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "read_file",
			Description: openai.String("Use this function to read and analyze a local file before modifying it."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]string{
						"type": "string",
					},
				},
				"required": []string{"file_path"},
			},
		},
	},
}

type ReadFileArguments struct {
	FilePath string `json:"file_path"`
}

func ReadFile(toolCall openai.ChatCompletionMessageToolCallUnion) openai.ChatCompletionMessageParamUnion {
	var args ReadFileArguments
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return openai.ToolMessage(fmt.Sprint("Error calling tool read_file():", err), toolCall.ID)
	}
	if args.FilePath == "" {
		return openai.ToolMessage(fmt.Sprint("Error calling tool read_file():", "parameter file_path is empty"), toolCall.ID)
	}

	f, err := os.Open(args.FilePath)
	if err != nil {
		return openai.ToolMessage(fmt.Sprint("Error calling tool read_file():", err), toolCall.ID)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return openai.ToolMessage(fmt.Sprint("Error calling tool read_file():", err), toolCall.ID)
	}

	return openai.ToolMessage(string(data), toolCall.ID)
}

var OverrideFileTool openai.ChatCompletionToolUnionParam = openai.ChatCompletionToolUnionParam{
	OfFunction: &openai.ChatCompletionFunctionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "override_file",
			Description: openai.String("Use this function to override a local file after identifying required changes."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]string{
						"type": "string",
					},
					"content": map[string]string{
						"type": "string",
					},
				},
				"required": []string{"file_path", "content"},
			},
		},
	},
}

type OverrideFileArguments struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func OverrideFile(toolCall openai.ChatCompletionMessageToolCallUnion) openai.ChatCompletionMessageParamUnion {
	var args OverrideFileArguments
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return openai.ToolMessage(fmt.Sprint("Error calling tool override_file:", err), toolCall.ID)
	}
	if args.FilePath == "" {
		return openai.ToolMessage(fmt.Sprint("Error calling tool override_file:", "argument file_path is empty"), toolCall.ID)
	}

	if err := os.WriteFile(args.FilePath, []byte(args.Content), 0644); err != nil {
		return openai.ToolMessage(fmt.Sprint("Error calling tool override_file:", err), toolCall.ID)
	}

	return openai.ToolMessage("File successfully overridden", toolCall.ID)
}

var ReplaceInFileTool openai.ChatCompletionToolUnionParam = openai.ChatCompletionToolUnionParam{
	OfFunction: &openai.ChatCompletionFunctionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "replace_in_file",
			Description: openai.String("Use this function to replace matching lines with other lines in a file. Make sure the search pattern only occurs exactly once."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]string{
						"type": "string",
					},
					"old_lines": map[string]string{
						"type": "string",
					},
					"new_lines": map[string]string{
						"type": "string",
					},
				},
				"required": []string{"file_path", "old_lines", "new_lines"},
			},
		},
	},
}

type ReplaceInFileArguments struct {
	FilePath    string `json:"file_path"`
	Pattern     string `json:"old_lines"`
	Replacement string `json:"new_lines"`
}

func ReplaceInFile(toolCall openai.ChatCompletionMessageToolCallUnion) openai.ChatCompletionMessageParamUnion {
	var args ReplaceInFileArguments
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return openai.ToolMessage(fmt.Sprint("Error calling tool replace_in_file():", err), toolCall.ID)
	}
	if args.FilePath == "" {
		return openai.ToolMessage(fmt.Sprint("Error calling tool replace_in_file():", "argument file_path is empty"), toolCall.ID)
	}
	if args.Pattern == "" {
		return openai.ToolMessage(fmt.Sprint("Error calling tool replace_in_file():", "argument pattern is empty"), toolCall.ID)
	}

	buf, err := os.ReadFile(args.FilePath)
	if err != nil {
		return openai.ToolMessage(fmt.Sprint("Error calling tool replace_in_file():", err), toolCall.ID)
	}

	text := string(buf)
	text = strings.ReplaceAll(text, args.Pattern, args.Replacement)

	err = os.WriteFile(args.FilePath, []byte(text), 0644)
	if err != nil {
		return openai.ToolMessage(fmt.Sprint("Error calling tool replace_in_file():", err), toolCall.ID)
	}

	return openai.ToolMessage("Text successfully replaced in file", toolCall.ID)
}
