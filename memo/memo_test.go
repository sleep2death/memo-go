package memo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMemo(t *testing.T) {
	m := New()
	require.NotEmpty(t, m.config.OpenAIAPIKey)
}

func TestPrompts(t *testing.T) {
	m := New()
	require.NotEmpty(t, m.config.OpenAIAPIKey)
	bootstrap := &bytes.Buffer{}
	err := m.config.Prompts.Bootstrap.Execute(bootstrap, map[string]string{
		"agent_name": "KD9-3.7",
	})
	require.NoError(t, err)
  t.Log("bootstrap:", bootstrap.String())
}
