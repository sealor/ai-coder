package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"golang.org/x/term"
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

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	t := term.NewTerminal(os.Stdin, "> ")

	for {
		prompt, err := t.ReadLine()
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(t, "Fatal:", err)
			}
			break
		}

		if prompt == "" {
			continue
		}

		param.Messages = append(param.Messages, openai.UserMessage(prompt))

		for {
			stream := client.Chat.Completions.NewStreaming(context.TODO(), param)
			acc, err := run(t, stream)
			if err != nil {
				fmt.Fprintln(t, "Fatal:", err)
				break
			}
			if err = stream.Close(); err != nil {
				fmt.Fprintln(t, "Fatal:", err)
				break
			}

			message := acc.Choices[0].Message
			param.Messages = append(param.Messages, message.ToParam())

			for _, toolCall := range message.ToolCalls {
				function := toolCall.Function
				if function.Name == "get_weather" {
					var args map[string]string
					json.Unmarshal([]byte(function.Arguments), &args)

					temperature := 10 + rand.Intn(15)
					SITUATION := []string{"rainy", "sunny", "cloudy"}
					sky := SITUATION[rand.Intn(len(SITUATION))]
					answer := fmt.Sprintf("Weather in %s: %d°C %s", args["city"], temperature, sky)

					param.Messages = append(param.Messages, openai.ToolMessage(answer, toolCall.ID))
					fmt.Fprintln(t, "Call result: ", answer)
				}
			}

			if len(message.ToolCalls) == 0 {
				break
			}
		}

		fmt.Fprintln(t, "")
	}
}

func run(w io.Writer, stream *ssestream.Stream[openai.ChatCompletionChunk]) (openai.ChatCompletionAccumulator, error) {
	acc := openai.ChatCompletionAccumulator{}

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		if content, ok := acc.JustFinishedContent(); ok {
			fmt.Fprintln(w, "Content stream finished:", content)
		}

		if tool, ok := acc.JustFinishedToolCall(); ok {
			fmt.Fprintln(w, "Tool call stream finished:", tool.Index, tool.Name, tool.Arguments)
		}

		if refusal, ok := acc.JustFinishedRefusal(); ok {
			fmt.Fprintln(w, "Refusal stream finished:", refusal)
		}

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			reasoningJSON, ok := choice.Delta.JSON.ExtraFields["reasoning"]
			var reasoning string
			if ok {
				json.Unmarshal([]byte(reasoningJSON.Raw()), &reasoning)
			}
			if len(reasoning) > 0 {
				fmt.Fprint(w, reasoning)
			}

			if len(choice.Delta.Content) > 0 {
				fmt.Fprint(w, choice.Delta.Content)
			}
		}
	}

	return acc, stream.Err()
}
