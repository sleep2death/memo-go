package memo

import (
	"context"
	"os"

	pb "github.com/qdrant/go-client/qdrant"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// setup mongodb
func SetupMongo() (*mongo.Collection, error) {
	dbUri := os.Getenv("MONGO_URI")
	if dbUri == "" {
		dbUri = "memo_db"
	}

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "memo_db"
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(dbUri))
	db := client.Database(dbName)

	dbSess := os.Getenv("MONGO_SESSIONS")
	if dbSess == "" {
		dbSess = "sessions"
	}

	return db.Collection(dbSess), err
}

// setup qdrant
func SetupQdrant() (pb.CollectionsClient, pb.PointsClient, error) {
	uri := os.Getenv("QDRANT_URI")
	if uri == "" {
		uri = "localhost:6334"
	}

	conn, err := grpc.Dial(uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	collections := pb.NewCollectionsClient(conn)
	points := pb.NewPointsClient(conn)

	return collections, points, nil
}
