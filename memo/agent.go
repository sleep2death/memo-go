package memo

import (
	"context"
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Agent struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string             `bson:"name" json:"name"`
}

const (
	AGENTS_COLLECTION = "agents"
)

// AddAgent to mongodb and create a collection in qdrant
func (m *Memo) AddAgent(ctx context.Context, agent Agent) (primitive.ObjectID, error) {
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

	_, err = m.EnsureQCollection(ctx, id.Hex())
	return id, err
}

// GetAgent from mongodb
func (m *Memo) GetAgent(ctx context.Context, id string) (agent *Agent, err error) {
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

	var ag Agent

	err = sr.Decode(&ag)
	return &ag, err
}

// DelAgent from mongodb and qdrant
func (m *Memo) DelAgent(ctx context.Context, id string) (err error) {
	coll := m.mongo.Database(m.config.MongoDb).Collection(AGENTS_COLLECTION)
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return
	}
	sr, err := coll.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return
	}

	// maybe docment not found
	if sr.DeletedCount == 0 {
		return fmt.Errorf("collection not found:%s", id)
	}

	// delete the related qdrant collection
	_, err = pb.NewCollectionsClient(m.qdrant).Delete(ctx, &pb.DeleteCollection{CollectionName: id})
	return
}

// ensure qdrant collection MUST exist, if not create one
func (m *Memo) EnsureQCollection(ctx context.Context, name string) (upsert bool, err error) {
	col := pb.NewCollectionsClient(m.qdrant)
	_, err = col.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: name})
	// already created
	if err == nil {
		return false, nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return false, err
	}

	// if collection not found, then create one
	if st.Code() == codes.NotFound {
		_, err := col.Create(ctx, &pb.CreateCollection{
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
		return true, err
	}
	return false, err
}
