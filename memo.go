package memo

import (
	"os"

	pb "github.com/qdrant/go-client/qdrant"
	openai "github.com/sashabaranov/go-openai"
	mongo "go.mongodb.org/mongo-driver/mongo"
)

type Handlers struct {
	sessions *mongo.Collection // mongodb collection for sessions

	qCollections pb.CollectionsClient // qdrant collection for memories
	qPoints      pb.PointsClient // qdrant points for memories

	openai *openai.Client // openai client

	SearchLimit int64 // search limit per page
}

func Default() (*Handlers, error) {
	sessions, err := SetupMongo()
	if err != nil {
		return nil, err
	}

	collection, points, err := SetupQdrant()
	if err != nil {
		return nil, err
	}

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, ErrInvalidOpenAPIKey
	}

  oc := openai.NewClient(key)

	return &Handlers{
		sessions:     sessions,
		qCollections: collection,
		qPoints:      points,
		openai:       oc,
		SearchLimit:  5,
	}, nil
}
