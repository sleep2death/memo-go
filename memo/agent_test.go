package memo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveAgent(t *testing.T) {
	m := New()
	m.config.MongoDb = "test-memo"

	ctx := context.TODO()
	// drop test database at last
	defer m.mongo.Database("test-memo").Drop(ctx)

	id, err := m.SaveAgent(ctx, Agent{Name: "Aspirin", Bootstrap: "this is a test bootstrap."})
	require.NoError(t, err)
	agent, err := m.GetAgent(ctx, id.Hex())
	require.Equal(t, agent.Name, "Aspirin")
	require.Equal(t, agent.Id, id)
}

func TestAgentBootstrap(t *testing.T) {
} 
