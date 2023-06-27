package memo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestSaveAgent(t *testing.T) {
	m := New()

	ctx := context.TODO()
	// drop test database at last
	defer m.mongo.Database(m.config.MongoDb).Drop(ctx)

	id, err := m.AddAgent(ctx, Agent{Name: "Aspirin"})
	require.NoError(t, err)

	agent, err := m.GetAgent(ctx, id.Hex())
	require.Equal(t, agent.Id, id)

	err = m.DelAgent(ctx, id.Hex())
	require.NoError(t, err)

	_, err = m.GetAgent(ctx, id.Hex())
	require.Equal(t, err, mongo.ErrNoDocuments)
}
