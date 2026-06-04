package main

import (
	"context"
	"fmt"
	"log"

	deepseek "github.com/cohesion-org/deepseek-go"
)

func main() {
	// Set up the Deepseek client
	client := deepseek.NewClient("sk-ad4a1acec470402cb8b0c63e2342d03c") // Empty API key triggers env lookup for "DEEPSEEK_API_KEY"

	// Create a chat completion request
	request := &deepseek.ChatCompletionRequest{
		Model: deepseek.DeepSeekV4Flash,
		Messages: []deepseek.ChatCompletionMessage{
			{Role: deepseek.ChatMessageRoleSystem, Content: "Answer every question using slang."},
			{Role: deepseek.ChatMessageRoleUser, Content: "你是谁"},
		},
	}

	// Send the request and handle the response
	ctx := context.Background()
	response, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Print the response
	fmt.Println("Response:", response.Choices[0].Message.Content)
}
