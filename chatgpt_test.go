package memo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreMemories(t *testing.T) {
	loadEnv(t)
	hs := Default()
	res, err := hs.llm.ScoreMemories(context.TODO(), hs.prompts.ScoreImportance, []string{"我今天早上吃了一顿麦当劳。", "昨天晚上我弄丢了无线耳机。"})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(res))

  // only one memory
	res, err = hs.llm.ScoreMemories(context.TODO(), hs.prompts.ScoreImportance, []string{"我今天下午可能要开会"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res))
}
