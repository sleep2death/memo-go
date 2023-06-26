package memo

import "github.com/sashabaranov/go-openai"

type promptsConfig struct {
	ScoreImportance []openai.ChatCompletionMessage `toml:"score_importance"`
}
