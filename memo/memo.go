package memo

import (
	"context"
	"text/template"

	"github.com/BurntSushi/toml"
	openai "github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ModeDebug   = "debug"
	ModeRelease = "release"
)

type Template struct {
	*template.Template
}

type Prompts struct {
	Bootstrap *Template
	Mail      *Template
}

func (t *Template) UnmarshalText(text []byte) error {
	template, err := new(template.Template).Parse(string(text))
	t.Template = template
	return err
}

type Config struct {
	// mongodb configs
	MongoUri string `toml:"mongo_uri"`
	MongoDb  string `toml:"mongo_db"` // mongodb database name

	QdrantUri string `toml:"qdrant_uri"` // qdrant grpc uri

	MemorySearchLimit     int `toml:"memory_search_limit"`
	MemorySearchThreshold float32 `toml:"memory_search_threshold"`

	// OPENAI
	OpenAIAPIKey string `toml:"openai_api_key"` // openai api auth key

	// Prompts
	Prompts Prompts `toml:"prompts"`

	// Memo Mode
	Mode string `toml:"mode"`
}

// the memo server
type Memo struct {
	mongo  *mongo.Client
	qdrant *grpc.ClientConn
	openai *openai.Client
	config *Config
}

// New will return a newly created memo server instance.
// it will load "config.toml" from current directory.
// make sure "openai_api_key" is in the config file, or it wil panic.
func New() *Memo {
	// load config file
	conf := Config{
		MongoUri:              "mongodb://localhost:27017",
		MongoDb:               "memo",
		QdrantUri:             "localhost:6334",
		Mode:                  "debug",
		MemorySearchLimit:     5,
		MemorySearchThreshold: 0.6,
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
	// create point index for memories
	col := mc.Database(conf.MongoDb).Collection(MEMORIES_COLLECTION)
	_, err = col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"pid": 1},
	})
	if err != nil {
		panic(err)
	}

	// qdrant
	qc, err := grpc.Dial(conf.QdrantUri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	// openai
	oc := openai.NewClient(conf.OpenAIAPIKey)

	return &Memo{
		config: &conf,
		openai: oc,
		mongo:  mc,
		qdrant: qc,
	}
}
