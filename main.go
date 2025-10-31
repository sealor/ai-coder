package main

import (
	"context"
	"encoding/json"
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
			openai.SystemMessage("Answer in exactly one short sentence!"),
		},
	}

	param.Messages = append(param.Messages, openai.UserMessage("Say this is a test"))

	stream := client.Chat.Completions.NewStreaming(context.TODO(), param)

	_, err := run(stream)
	if err != nil {
		panic(err)
	}
}

func run(stream *ssestream.Stream[openai.ChatCompletionChunk]) (openai.ChatCompletionAccumulator, error) {
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

	return acc, stream.Err()
}
