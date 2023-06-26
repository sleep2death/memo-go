package memo

import (
	"context"

	"github.com/BurntSushi/toml"
	openai "github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	// mongodb configs
	MongoUri string `toml:"mongo_uri"`
	MongoDb  string `toml:"mongo_db"` // mongodb database name

	QdrantUri string `toml:"qdrant_uri"` // qdrant grpc uri

	// OPENAI
	OpenAIAPIKey string `toml:"openai_api_key"` // openai api auth key
}

// the memo server
type Memo struct {
	mongo  *mongo.Client
	qdrant *grpc.ClientConn
	openai *openai.Client
	config Config
}

// New will return a newly created memo server instance.
// it will load "config.toml" from current directory.
// make sure "openai_api_key" is in the config file, or it wil panic.
func New() *Memo {
	// load config file
	conf := Config{
		MongoUri:  "mongodb://localhost:27017",
		MongoDb:   "memo",
		QdrantUri: "localhost:6334",
	}

	_, err := toml.DecodeFile("config.toml", &conf)
	if err != nil {
		panic(err)
	}

	// openai
	if conf.OpenAIAPIKey == "" {
		panic("need openai api key to continue")
	}

	ctx := context.TODO()

	// mongodb
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(conf.MongoUri))
	if err != nil {
		panic(err)
	}

	// qdrant
	qc, err := grpc.Dial(conf.QdrantUri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	return &Memo{
		config: conf,
		openai: openai.NewClient(conf.OpenAIAPIKey),
		mongo:  mc,
		qdrant: qc,
	}
}
