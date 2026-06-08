package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cohesion-org/deepseek-go"
	"github.com/wtitdn/renew_video/internal/config"
	"github.com/wtitdn/renew_video/internal/entity"
)

type LLMClient struct {
	client *deepseek.Client
}

func NewClient(cfg *config.ApiKeyConfig) (*LLMClient, error) {
	cilent := deepseek.NewClient(cfg.Apikey)
	if cilent != nil {
		return &LLMClient{client: cilent}, nil
	}
	return nil, errors.New("Invalid API key")
}

func (l *LLMClient) LLMSummary(ctx context.Context, comments []entity.Comment) (string, error) {
	var builder strings.Builder
	limit := 3
	if len(comments) < limit {
		limit = len(comments)
	}

	for i, comment := range comments[:limit] {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, comment.Content))
	}

	prompt := fmt.Sprintf(`请总结下面这些评论，提炼主要观点、用户情绪和高频问题：%s`, builder.String())

	request := &deepseek.ChatCompletionRequest{
		Model: deepseek.DeepSeekV4Flash,
		Messages: []deepseek.ChatCompletionMessage{
			{
				Role:    deepseek.ChatMessageRoleSystem,
				Content: "你是一个评论区总结助手，请用简洁中文总结用户评论。",
			},
			{
				Role:    deepseek.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	response, err := l.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "llm调用失败", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("empty llm response")
	}
	return response.Choices[0].Message.Content, nil
}
func (l *LLMClient) Close() error {
	return nil
}
