package main

import (
	"context"
	"fmt"
	"log"
	"os"

	deepseek "github.com/cohesion-org/deepseek-go"
	"github.com/wtitdn/renew_video/internal/config"
)

func main() {
	// Set up the Deepseek client
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "../config/config.yaml"
	}
	cfg, _, err := config.LoadLocalDev(configPath)
	apikey := cfg.ApiKeyConfig.Apikey
	print(apikey)
	client := deepseek.NewClient(apikey) // Empty API key triggers env lookup for "DEEPSEEK_API_KEY"

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
