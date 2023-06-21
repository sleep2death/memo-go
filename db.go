package memo

import (
	"context"
	"errors"
	"os"

	pb "github.com/qdrant/go-client/qdrant"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// setup mongodb
func SetupMongo() (*mongo.Database, error) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return nil, errors.New("'MONGO_URI' not found in enviroment")
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	db := client.Database(os.Getenv("MONGO_DB"))
	return db, err
}

// setup qdrant
func SetupQdrant() (pb.CollectionsClient, pb.PointsClient, error) {
	uri := os.Getenv("QDRANT_URI")
	if uri == "" {
		return nil, nil, errors.New("'QDRANT_URI' not found in enviroment")
	}

	conn, err := grpc.Dial(uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	collections := pb.NewCollectionsClient(conn)
	points := pb.NewPointsClient(conn)

	return collections, points, nil
}
