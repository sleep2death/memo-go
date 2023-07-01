package memo

import (
	"fmt"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
)

type Session struct {
	mongo  *mongo.Collection
	qdrant pb.CollectionsClient
	ctx    context.Context
	limit  int64
}

// Add agent and return inserted id
// if agent's id is not set, then it will create one
// if agent's created time is not set, then it will use "time.now"
func (s *Session) AddAgent(agent *Agent) (primitive.ObjectID, error) {
	if agent.ID == primitive.NilObjectID {
		agent.ID = primitive.NewObjectID()
	}

	if agent.Created.IsZero() {
		agent.Created = time.Now()
	}

	_, err := s.mongo.InsertOne(s.ctx, agent)
	if err != nil {
		return primitive.NilObjectID, err
	}

	err = s.createQdrantCollection(agent.ID.Hex())
	if err != nil {
		return primitive.NilObjectID, err
	}
	return agent.ID, nil
}

// Delete agent, if no agent matched it will return an notfound error
func (s Session) DeleteAgent(id primitive.ObjectID) error {
	res, err := s.mongo.DeleteOne(s.ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return fmt.Errorf("can't find the agent: %s", id)
	}

	return s.deleteQdrantCollection(id.Hex())
}

// Update agent, if no agent matched it will return an notfound error
func (s *Session) UpdateAgent(agent *Agent) error {
	res, err := s.mongo.UpdateByID(s.ctx, agent.ID, bson.M{"$set": agent})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("can't find the agent: %s", agent.ID.Hex())
	}

	return err
}

// GetAgent by id
func (s *Session) GetAgent(id primitive.ObjectID) (agent *Agent, err error) {
	res := s.mongo.FindOne(s.ctx, bson.M{"_id": id})
	err = res.Err()
	if err != nil {
		return
	}

	agent = &Agent{}
	err = res.Decode(agent)
	return
}

// List agents with offset, you can set search limit by session
func (s *Session) ListAgents(offset primitive.ObjectID) (agents []*Agent, err error) {
	opts := options.Find().SetSort(bson.M{"_id": -1}).SetLimit(s.limit)
	var filter bson.M
	// if offset is not nil, then make the offset filter
	if offset != primitive.NilObjectID {
		filter = bson.M{"_id": bson.M{"$lt": offset}}
	}
	cur, err := s.mongo.Find(s.ctx, filter, opts)

	if err != nil {
		return
	}
	err = cur.All(s.ctx, &agents)
	return
}

func (s Session) createQdrantCollection(name string) (err error) {
	_, err = s.qdrant.Create(s.ctx, &pb.CreateCollection{
		CollectionName: name,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     1536,
					Distance: pb.Distance_Cosine,
				},
			},
		},
	})
	return
}

func (s Session) deleteQdrantCollection(name string) (err error) {
	_, err = s.qdrant.Delete(s.ctx, &pb.DeleteCollection{CollectionName: name})
	return
}
