package memo

import (
	pb "github.com/qdrant/go-client/qdrant"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handlers struct {
	mongo *mongo.Database

	qCollections pb.CollectionsClient
	qPoints      pb.PointsClient

	SearchLimit int64
}

func New() (*Handlers, error) {
	mongo, err := SetupMongo()
	if err != nil {
		return nil, err
	}

	c, p, err := SetupQdrant()
	if err != nil {
		return nil, err
	}

	return &Handlers{
		mongo:        mongo,
		qCollections: c,
		qPoints:      p,
		SearchLimit:  5,
	}, nil
}
