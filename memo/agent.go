package memo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Agent struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	Bootstrap string             `bson:"boot" json:"boot"` // the bootstrap sentences to start an agent
}


const (
	AGENTS_COLLECTION = "agents"
)

// SaveAgent to mongodb
func (m *Memo) SaveAgent(ctx context.Context, agent Agent) (primitive.ObjectID, error) {
	if agent.Id == primitive.NilObjectID {
		agent.Id = primitive.NewObjectID()
	}

	coll := m.mongo.Database(m.config.MongoDb).Collection(AGENTS_COLLECTION)
	opts := options.Update().SetUpsert(true)
	res, err := coll.UpdateByID(ctx, agent.Id, bson.M{"$set": agent}, opts)
	if err != nil {
		return primitive.NilObjectID, err
	}

	id, ok := res.UpsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("invalid objectid: %v", res.UpsertedID)
	}

	return id, nil
}

// GetAgent from mongodb
func (m *Memo) GetAgent(ctx context.Context, id string) (agent Agent, err error) {
	coll := m.mongo.Database(m.config.MongoDb).Collection(AGENTS_COLLECTION)
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return
	}
	sr := coll.FindOne(ctx, bson.M{"_id": oid})
	// maybe docment not found
	if err = sr.Err(); err != nil {
		return
	}

	err = sr.Decode(&agent)
	return
}

// Bootstrap will generate the first message of this agent
func (m *Memo) Bootstrap(agent Agent) {
}
