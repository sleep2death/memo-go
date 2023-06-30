package memo

import (
	"context"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type llm struct {
	client *openai.Client
}

func (l *llm) ScoreMemories(ctx context.Context, prompts []openai.ChatCompletionMessage, memories []string) (scores []int, err error) {
  questions := openai.ChatCompletionMessage{Role: "user", Content: strings.Join(memories, ";")}

	resp, err := l.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: append(prompts, questions),
	})

	if err != nil {
		return nil, err
	}
	return sliceAtoi(strings.Split(resp.Choices[0].Message.Content, ", "))
}

// create embeddings from openai
func (l *llm) embedding(ctx context.Context, input []string) (openai.EmbeddingResponse, error) {
	erq := openai.EmbeddingRequest{
		Input: input,
		Model: openai.AdaEmbeddingV2,
	}
	return l.client.CreateEmbeddings(ctx, erq)
}

func sliceAtoi(sa []string) ([]int, error) {
	si := make([]int, 0, len(sa))
	for _, a := range sa {
		i, err := strconv.Atoi(a)
		if err != nil {
			return si, err
		}
		si = append(si, i)
	}
	return si, nil
}
