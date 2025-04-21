package main

import (
	"agent/tools"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/anthropics/anthropic-sdk-go"
	"os"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
}

func NewAgent(client *anthropic.Client, getUserMessage func() (string, bool), tools []tools.ToolDefinition) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
	}
}

func (agent *Agent) RunInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	var anthropicTools []anthropic.ToolUnionParam
	for _, tool := range agent.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	message, err := agent.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		MaxTokens: int64(1024),
		Messages:  conversation,
		Tools:     anthropicTools,
	})
	return message, err
}

func (agent *Agent) Run(ctx context.Context) error {
	var conversation []anthropic.MessageParam

	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	readUserInput := true

	// Agent loop
	for {
		if readUserInput {
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := agent.getUserMessage()
			if !ok {
				break
			}
			userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
			conversation = append(conversation, userMessage)
		}

		message, err := agent.RunInference(ctx, conversation)
		if err != nil {
			return fmt.Errorf("failed to get response: %w", err)
		}
		conversation = append(conversation, message.ToParam())

		var toolResults []anthropic.ContentBlockParamUnion
		for _, content := range message.Content {
			switch content.Type {
			case "text":
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)

			case "tool_use":
				result := agent.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}
		if len(toolResults) == 0 {
			readUserInput = true
		} else {
			readUserInput = false
			conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
		}
	}

	return nil
}

func (agent *Agent) executeTool(toolID string, toolName string, input json.RawMessage) anthropic.ContentBlockParamUnion {

	for _, tool := range agent.tools {
		if tool.Name == toolName {
			fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", toolName, input)
			result, err := tool.Function(input)
			if err != nil {
				fmt.Printf("Error executing tool %s: %v\n", toolName, err)
				return anthropic.NewToolResultBlock(toolID, err.Error(), true)
			}
			return anthropic.NewToolResultBlock(toolID, result, false)
		}
	}
	return anthropic.NewTextBlock(fmt.Sprintf("Tool %s not found", toolName))
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

	// Define tools here if needed
	anthropicTools := []tools.ToolDefinition{tools.ReadFileDefinition, tools.ListFilesDefinition, tools.EditFileDefinition}
	agent := NewAgent(&client, getUserMessage, anthropicTools)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error running agent: %v\n", err)
	}
}
