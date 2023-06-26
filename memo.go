package memo

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	pb "github.com/qdrant/go-client/qdrant"
	openai "github.com/sashabaranov/go-openai"
	mongo "go.mongodb.org/mongo-driver/mongo"
)

type Handlers struct {
	sessions *mongo.Collection // mongodb collection for sessions

	qCollections pb.CollectionsClient // qdrant collection for memories
	qPoints      pb.PointsClient      // qdrant points for memories

	llm *llm // openai wrapper

	SearchLimit int64         // search limit per page
	prompts     promptsConfig // prompts config
}

func Default() *Handlers {
	sessions, err := SetupMongo()
	if err != nil {
		panic(err)
	}

	collection, points, err := SetupQdrant()
	if err != nil {
		panic(err)
	}

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		panic(ErrInvalidOpenAPIKey)
	}

	llm := &llm{
		client: openai.NewClient(key),
	}

	// read prompts config
	cfg, err := os.ReadFile("prompts.toml")
	if err != nil {
		panic(fmt.Errorf("fatal error load prompts config file: %w", err))
	}

	var prompts promptsConfig
	toml.Decode(string(cfg), &prompts)
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("fatal error parse prompts config file: %w", err))
	}

	hs := &Handlers{
		sessions:     sessions,
		qCollections: collection,
		qPoints:      points,
		llm:          llm,
		prompts:      prompts,
		SearchLimit:  5,
	}

	return hs
}
