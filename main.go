package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"syscall"

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
	apiURL := flag.String("api", GetEnv("OPENAI_URL", "http://127.0.0.1:11434/v1"), "URL for the OpenAI API endpoint")
	model := flag.String("model", "qwen3:1.7b", "Technical name of the LLM")
	activeLog := flag.Bool("log", false, "Activates logging")
	message := flag.String("message", "", "User message")

	flag.Parse()

	options := []option.RequestOption{
		option.WithBaseURL(*apiURL),
	}
	if *activeLog {
		options = append(options, option.WithDebugLog(nil))
	}
	client := openai.NewClient(options...)

	param := openai.ChatCompletionNewParams{
		Model:           *model,
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

	t := term.NewTerminal(os.Stdin, "> ")

	for {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Fprintln(t, "Fatal:", err)
			break
		}
		prompt := *message
		if len(*message) == 0 {
			prompt, err = t.ReadLine()
		}
		restoreErr := term.Restore(int(os.Stdin.Fd()), oldState)

		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(t, "Fatal:", err)
			}
			break
		}
		if restoreErr != nil {
			fmt.Fprintln(t, "Fatal:", restoreErr)
			break
		}

		if prompt == "" {
			continue
		}

		param.Messages = append(param.Messages, openai.UserMessage(prompt))

		runPrompt(t, client, &param)

		if len(*message) > 0 {
			break
		}
	}
}

func runPrompt(w io.Writer, client openai.Client, param *openai.ChatCompletionNewParams) {
	for {
		stream := client.Chat.Completions.NewStreaming(context.TODO(), *param)
		acc, err := run(w, stream)
		if err != nil {
			fmt.Fprintln(w, "Fatal:", err)
			break
		}
		if err = stream.Close(); err != nil {
			fmt.Fprintln(w, "Fatal:", err)
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
				fmt.Fprintln(w, "Call result: ", answer)
			}
		}

		if len(message.ToolCalls) == 0 {
			break
		}
	}

	fmt.Fprintln(w, "")
}

func run(w io.Writer, stream *ssestream.Stream[openai.ChatCompletionChunk]) (openai.ChatCompletionAccumulator, error) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	acc := openai.ChatCompletionAccumulator{}

loop:
	for stream.Next() {
		select {
		case <-ctx.Done():
			break loop
		default:
		}

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
