package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
)

func GetEnv(name, fallback string) string {
	value, ok := os.LookupEnv(name)
	if ok {
		return value
	} else {
		return fallback
	}
}

func main() {
	println("Go!")

	client := openai.NewClient(
		option.WithBaseURL(GetEnv("OLLAMA_URL", "http://nixos.lan:11434/v1")),
		// option.WithDebugLog(nil),
	)

	param := openai.ChatCompletionNewParams{
		Model:           "qwen3:1.7b",
		ReasoningEffort: "none",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("Answer using provided tools"),
		},
		Tools: []openai.ChatCompletionToolUnionParam{
			{
				OfFunction: &openai.ChatCompletionFunctionToolParam{
					Function: openai.FunctionDefinitionParam{
						Name:        "get_weather",
						Description: openai.String("Get local weather"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"city": map[string]string{
									"type": "string",
								},
							},
							"required": []string{"city"},
						},
					},
				},
			},
		},
	}

	param.Messages = append(param.Messages, openai.UserMessage("Get weather of Leipzig"))

	stream := client.Chat.Completions.NewStreaming(context.TODO(), param)
	if _, err := run(stream, &param); err != nil {
		panic(err)
	}

	stream = client.Chat.Completions.NewStreaming(context.TODO(), param)
	if _, err := run(stream, &param); err != nil {
		panic(err)
	}
}

func run(stream *ssestream.Stream[openai.ChatCompletionChunk], param *openai.ChatCompletionNewParams) (openai.ChatCompletionAccumulator, error) {
	acc := openai.ChatCompletionAccumulator{}

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		if content, ok := acc.JustFinishedContent(); ok {
			println("Content stream finished:", content)
		}

		if tool, ok := acc.JustFinishedToolCall(); ok {
			println("Tool call stream finished:", tool.Index, tool.Name, tool.Arguments)
		}

		if refusal, ok := acc.JustFinishedRefusal(); ok {
			println("Refusal stream finished:", refusal)
		}

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			reasoningJSON, ok := choice.Delta.JSON.ExtraFields["reasoning"]
			var reasoning string
			if ok {
				json.Unmarshal([]byte(reasoningJSON.Raw()), &reasoning)
			}
			if len(reasoning) > 0 {
				print(reasoning)
			}

			if len(choice.Delta.Content) > 0 {
				print(choice.Delta.Content)
			}
		}
	}

	message := acc.Choices[0].Message
	param.Messages = append(param.Messages, message.ToParam())

	for _, toolCall := range message.ToolCalls {
		function := toolCall.Function
		if function.Name == "get_weather" {
			var args map[string]string
			json.Unmarshal([]byte(function.Arguments), &args)
			answer := fmt.Sprintf("Weather in %s: 28°C sunny", args["city"])
			param.Messages = append(param.Messages, openai.ToolMessage(answer, toolCall.ID))
		}
	}

	return acc, stream.Err()
}
