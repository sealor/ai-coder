package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/shared"
	"github.com/sealor/ai-coder/pkg/persistence"
	"github.com/sealor/ai-coder/pkg/tooling"
	"golang.org/x/term"
)

// Provide exactly one recommendation for a refactoring as a comment AI!

var toolFuncByName = map[string]func(openai.ChatCompletionMessageToolCallUnion) openai.ChatCompletionMessageParamUnion{
	"read_file":       tooling.ReadFile,
	"override_file":   tooling.OverrideFile,
	"replace_in_file": tooling.ReplaceInFile,
}

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
	userMessage := flag.String("message", "", "User message")
	systemMessage := flag.String("system", "", "System message")
	reasoning := flag.String("reasoning", "", "Level of reasoning (e.g. none, low, medium, high)")
	sessionFile := flag.String("session-file", "", "Use this file to save and resume chat sessions")
	activateTools := flag.Bool("tools", false, "Activate tools")
	activeLog := flag.Bool("log", false, "Activate logging")

	flag.Parse()

	options := []option.RequestOption{
		option.WithBaseURL(*apiURL),
	}
	apiKey := GetEnv("OPENAI_API_KEY", "")
	if apiKey != "" {
		options = append(options, option.WithAPIKey(apiKey))
	}
	if *activeLog {
		options = append(options, option.WithDebugLog(nil))
	}
	client := openai.NewClient(options...)

	param := openai.ChatCompletionNewParams{}
	if *sessionFile != "" {
		var err error
		param, err = persistence.TryToResumeSession(*sessionFile)
		if err != nil {
			log.Fatalln("ERROR:", err)
		}
	}

	if *model != "" {
		param.Model = *model
	}

	if *reasoning != "" {
		param.ReasoningEffort = shared.ReasoningEffort(*reasoning)
	}

	if *systemMessage != "" {
		param.Messages = append(param.Messages, openai.SystemMessage(*systemMessage))
	}

	if *activateTools {
		param.Tools = []openai.ChatCompletionToolUnionParam{
			tooling.ReadFileTool, tooling.OverrideFileTool, tooling.ReplaceInFileTool,
		}
	}

	t := term.NewTerminal(os.Stdin, "> ")

	for {
		prompt := *userMessage
		if len(*userMessage) == 0 {
			fd := int(os.Stdin.Fd())
			oldState, err := term.MakeRaw(fd)
			if err != nil {
				fmt.Fprintln(t, "Fatal:", err)
				break
			}

			width, height, err := term.GetSize(fd)
			if err != nil {
				fmt.Fprintln(t, "Fatal:", err)
				break
			}
			t.SetSize(width, height)

			prompt, err = t.ReadLine()
			restoreErr := term.Restore(fd, oldState)

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
		}

		if prompt == "" {
			continue
		}

		param.Messages = append(param.Messages, openai.UserMessage(prompt))

		runPrompt(t, client, &param)

		if *sessionFile != "" {
			if err := persistence.SaveSession(*sessionFile, &param); err != nil {
				log.Fatalln("ERROR:", err)
			}
		}

		if len(*userMessage) > 0 {
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
			toolFunc, ok := toolFuncByName[toolCall.Function.Name]
			if ok {
				fmt.Fprintln(w, "Tool Call:", toolCall.Function)
				message := toolFunc(toolCall)
				param.Messages = append(param.Messages, message)
				fmt.Fprintln(w, "Result:", message.OfTool.Content)
			} else {
				fmt.Fprintln(w, "Function for Tool Call missing:", toolCall.Function)
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
