package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/anthropics/anthropic-sdk-go"
	"os"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
}

func NewAgent(client *anthropic.Client, getUserMessage func() (string, bool)) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
	}
}

func main() {
	client := anthropic.NewClient()
	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if scanner.Scan() {
			return scanner.Text(), true
		}
		return "", false
	}

	agent := NewAgent(&client, getUserMessage)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error running agent: %v\n", err)
	}
}

func (agent *Agent) RunInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	message, err := agent.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		MaxTokens: int64(1024),
		Messages:  conversation,
	})
	return message, err
}

func (agent *Agent) Run(ctx context.Context) error {
	var conversation []anthropic.MessageParam

	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	// Agent loop
	for {
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := agent.getUserMessage()
		if !ok {
			break
		}

		userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
		conversation = append(conversation, userMessage)
		message, err := agent.RunInference(ctx, conversation)
		if err != nil {
			return fmt.Errorf("failed to get response: %w", err)
		}
		conversation = append(conversation, message.ToParam())

		for _, content := range message.Content {
			switch content.Type {
			case "text":
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			}
		}
	}

	return nil
}
