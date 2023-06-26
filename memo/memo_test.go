package memo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	m := New()
	assert.NotEmpty(t, m.config.OpenAIAPIKey)
}
