package memo

import "go.mongodb.org/mongo-driver/bson/primitive"

type AgentModel interface {
	// Add agent and return inserted id
	AddAgent(agent Agent) (primitive.ObjectID, error)

	// Delete agent by id
	DeleteAgent(id primitive.ObjectID) error

	// Update agent
	UpdateAgent(agent Agent) error

	// ListAgents and offset agent's id
	ListAgents(offset primitive.ObjectID) ([]Agent, error)

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
