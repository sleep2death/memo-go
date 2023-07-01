package memo

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Agent struct {
	ID      primitive.ObjectID `bson:"_id" json:"id"`
	Name    string             `bson:"name" json:"name"`
	Created time.Time          `bson:"created_at" json:"created_at"`
}

type Memory struct {
	ID      primitive.ObjectID `bson:"_id" json:"id"`
	Content string             `bson:"content" json:"content"`
	PID     string             `bson:"pid" json:"pid"`
	Created time.Time          `bson:"created_at" json:"created_at"`
}

type AgentModel interface {
	// Add agent and return inserted id
	AddAgent(agent Agent) (string, error)

	// Delete agent by id
	DeleteAgent(id primitive.ObjectID) error

	// Update agent
	UpdateAgent(agent Agent) error

	// ListAgents and offset agent's id
	ListAgents(offset primitive.ObjectID) ([]Agent, string, error)

	// GetAgent by id
	GetAgent(id primitive.ObjectID) (Agent, error)
}

type MemoryModel interface {
	// Add memory and return inserted id
	AddMemory(memory Memory) (string, error)
	// Add memories and return inserted ids
	AddMemories(memories []Memory) ([]string, error)

	// Update memory
	UpdateMemory(memory Memory) error
	// Update memories
	UpdateMemories(memories []Memory) error

	// Delete memory by id
	DeleteMemory(id primitive.ObjectID) error
	// Delete memories by ids
	DeleteMemories(ids []primitive.ObjectID) error

	// ListMemories and offset memory's id
	ListMemories(offset primitive.ObjectID) ([]Memory, string, error)

	// Search related memories by query string,
	Search(query string, limit string) ([]Memory, error)
}
