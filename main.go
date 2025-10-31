package main

import (
	"context"
	"os"

	"github.com/openai/openai-go/packages/ssestream"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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
		// option.WithDebugLog(nil)
	)

	param := openai.ChatCompletionNewParams{
		Model: "qwen3:1.7b",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("Answer in exactly one short sentence!"),
		},
	}

	param.Messages = append(param.Messages, openai.UserMessage("Say this is a test"))

	completion, err := client.Chat.Completions.New(context.TODO(), param)
	if err != nil {
		panic(err.Error())
	}

	println(completion.Choices[0].Message.Content)
}
