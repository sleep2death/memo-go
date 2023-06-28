package memo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	openai "github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	MEMORIES_COLLECTION = "memories"
)

type Memory struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	Content   string             `bson:"content" json:"content"`
	PID       string             `bson:"pid" json:"pid"`
}

func (memo *Memo) AddMemories(ctx context.Context, agent *Agent, memories []*Memory) error {
	var contents []string
	for _, m := range memories {
		m.ID = primitive.NewObjectID() // generate new objectid
		if m.CreatedAt.IsZero() {
			m.CreatedAt = time.Now()
		}

		contents = append(contents, m.Content)
	}

	// create embeddings
	emreq := openai.EmbeddingRequest{
		Input: contents,
		Model: openai.AdaEmbeddingV2,
	}

	emres, err := memo.openai.CreateEmbeddings(ctx, emreq)
	if err != nil {
		return err
	}

	// create points which will be inserted into qdrant
	points := make([]*pb.PointStruct, len(memories))
	for _, em := range emres.Data {
		uuid, _ := uuid.NewUUID() // using uuid v1 which contains timestamp info

		point := &pb.PointStruct{
			Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: uuid.String()}},
			Payload: map[string]*pb.Value{"mongo": {Kind: &pb.Value_StringValue{StringValue: memories[em.Index].ID.Hex()}}},
			Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: em.Embedding}}},
		}
		points[em.Index] = point
		memories[em.Index].PID = uuid.String() // save pid
	}

	waitUpsert := true
	_, err = pb.NewPointsClient(memo.qdrant).Upsert(ctx, &pb.UpsertPoints{
		CollectionName: agent.Id.Hex(),
		Wait:           &waitUpsert,
		Points:         points,
	})

	if err != nil {
		return err
	}

	// save memories into mongodo collection
	col := memo.mongo.Database(memo.config.MongoDb).Collection(MEMORIES_COLLECTION)

	var docs []interface{}
	for _, m := range memories {
		docs = append(docs, m)
	}
	_, err = col.InsertMany(ctx, docs)
	return err
}

func (memo *Memo) DeleteMemories(ctx context.Context, agent *Agent, memories []*Memory) error {
	l := len(memories)
	mids := make([]primitive.ObjectID, l)
	pids := make([]*pb.PointId, l)

	for idx, m := range memories {
		mids[idx] = m.ID
		pids[idx] = &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: m.PID}}
	}

	// delete from mongodb
	col := memo.mongo.Database(memo.config.MongoDb).Collection(MEMORIES_COLLECTION)
	res, err := col.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": mids}})
	if err != nil {
		return err
	}

	if res.DeletedCount != int64(l) {
		return fmt.Errorf("some memories not found")
	}

	// delete from qdrant
	pb.NewPointsClient(memo.qdrant).Delete(ctx, &pb.DeletePoints{
		CollectionName: agent.Id.Hex(),
		Points:         &pb.PointsSelector{PointsSelectorOneOf: &pb.PointsSelector_Points{Points: &pb.PointsIdsList{Ids: pids}}},
	})

	return nil
}

func (memo *Memo) SearchMemories(ctx context.Context, agent *Agent, query string) ([]*Memory, []float32, error) {
	// create embeddings
	emreq := openai.EmbeddingRequest{
		Input: []string{query},
		Model: openai.AdaEmbeddingV2,
	}

	emres, err := memo.openai.CreateEmbeddings(ctx, emreq)
	if err != nil {
		return nil, nil, err
	}

	// search points
	res, err := pb.NewPointsClient(memo.qdrant).Search(ctx, &pb.SearchPoints{
		CollectionName: agent.Id.Hex(),
		Vector:         emres.Data[0].Embedding,
		Limit:          uint64(memo.config.MemorySearchLimit),
		ScoreThreshold: &memo.config.MemorySearchThreshold,
		// will not include vectors
		WithVectors: &pb.WithVectorsSelector{SelectorOptions: &pb.WithVectorsSelector_Enable{Enable: false}},
		// will include metadata
		WithPayload: &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})

	// turn mongo reference into objectid, and save scores
	var scores []float32
	var mids []primitive.ObjectID
	for _, point := range res.Result {
		payload := point.GetPayload()
		mid := payload["mongo"].GetStringValue()
		if mid == "" {
			return nil, nil, fmt.Errorf("point's mongo collection reference is empty")
		}

		oid, err := primitive.ObjectIDFromHex(mid)
		if err != nil {
			return nil, nil, err
		}
		mids = append(mids, oid)
		scores = append(scores, point.GetScore())
	}

	// search for them
	q := bson.M{"_id": bson.M{"$in": mids}}
	col := memo.mongo.Database(memo.config.MongoDb).Collection(MEMORIES_COLLECTION)
	cur, err := col.Find(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	var docs []*Memory
	err = cur.All(ctx, &docs)

	// sort
	sorted := make([]*Memory, len(mids))
	for idx, mid := range mids {
		for _, m := range docs {
			if mid == m.ID {
				sorted[idx] = m
			}
		}
	}

	return sorted, scores, err
}
