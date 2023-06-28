package memo

import (
	"context"
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func TestAddMemories(t *testing.T) {
	m := New()
	ctx := context.TODO()
	// drop test database at last
	defer m.mongo.Database(m.config.MongoDb).Drop(ctx)

	id, err := m.AddAgent(ctx, Agent{Name: "aspirin"})
	assert.NoError(t, err)

	agent, err := m.GetAgent(ctx, id.Hex())
	assert.NoError(t, err)

	err = m.AddMemories(ctx, agent, []*Memory{
		{Content: "我的名字叫明日香。"},
		{Content: "我今年14岁。"},
		{Content: "我是德日混血，国籍是美国。"},
	})
	assert.NoError(t, err)

	res, _, err := m.SearchMemories(ctx, agent, "你的国籍是哪里？")
	assert.NoError(t, err)
	assert.Contains(t, res[0].Content, "美国")

	res, _, err = m.SearchMemories(ctx, agent, "你的年龄")
	assert.NoError(t, err)
	assert.Contains(t, res[0].Content, "14")

  err = m.DeleteMemories(ctx, agent, res)
	assert.NoError(t, err)

	res, _, err = m.SearchMemories(ctx, agent, "你的国籍是哪里？")
	assert.NoError(t, err)
	assert.Zero(t, len(res))

	// delete agent at the end
	m.DelAgent(ctx, id.Hex())
}
