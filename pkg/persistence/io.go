package persistence

import (
	"os"

	"github.com/openai/openai-go/v3"
	"gopkg.in/yaml.v3"
)

func SaveSession(sessionFile string, param *openai.ChatCompletionNewParams) error {
	session := NewSessionFromOpenAI(param)
	data, err := yaml.Marshal(session)
	if err != nil {
		return err
	}
	if err = os.WriteFile(sessionFile, data, 0640); err != nil {
		return err
	}
	return nil
}

func TryToResumeSession(sessionFile string) (openai.ChatCompletionNewParams, error) {
	_, err := os.Stat(sessionFile)
	if os.IsNotExist(err) {
		return openai.ChatCompletionNewParams{}, nil
	}
	if err != nil {
		return openai.ChatCompletionNewParams{}, err
	}

	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return openai.ChatCompletionNewParams{}, err
	}

	var session Session
	if err = yaml.Unmarshal(data, &session); err != nil {
		return openai.ChatCompletionNewParams{}, err
	}

	return *NewParamsFromSession(&session), nil
}
