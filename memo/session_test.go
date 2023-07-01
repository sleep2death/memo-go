package memo

import (
	"fmt"
	"testing"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SessionSuite struct {
	suite.Suite
	session Session
}

func (s *SessionSuite) SetupSuite() {
	ctx := context.TODO()
	// mongodb
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	// qdrant
	qc, err := grpc.Dial("localhost:6334", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	s.session = Session{
		ctx:    ctx,
		qdrant: pb.NewCollectionsClient(qc),
		mongo:  mc.Database("test-db").Collection("agents"),
		limit:  15,
	}
}

func (s *SessionSuite) TearDownTest() {
	// drop agent collection when each test finished
	err := s.session.mongo.Drop(s.session.ctx)
	s.NoError(err)
}
func (s *SessionSuite) TearDownSuite() {
	// delete all qdrant collections when all tests in this suite finished
	res, err := s.session.qdrant.List(s.session.ctx, &pb.ListCollectionsRequest{})
	s.NoError(err)
	for _, col := range res.Collections {
		_, err := s.session.qdrant.Delete(s.session.ctx, &pb.DeleteCollection{CollectionName: col.Name})
		s.NoError(err)
	}
}

func (s *SessionSuite) TestAddAgent() {
	id0, err := s.session.AddAgent(Agent{Name: "aspirin"})
	s.NoError(err)
	id1, err := s.session.AddAgent(Agent{Name: "aspirin2d"})
	s.NoError(err)

	s.NotEqual(id0.Hex(), id1.Hex())

	newID := primitive.NewObjectID()
	id3, err := s.session.AddAgent(Agent{ID: newID, Name: "aspirin2d"})
	s.NoError(err)
	s.Equal(newID.Hex(), id3.Hex())
}

func (s *SessionSuite) TestGetAgent() {
	id, err := s.session.AddAgent(Agent{Name: "aspirin"})
	s.NoError(err)
	agent, err := s.session.GetAgent(id)
	s.NoError(err)
	s.Equal(agent.Name, "aspirin")
	// it will create a new "created" value for the agent
	s.True(agent.Created.After(time.Now().Add(-5 * time.Second)))

	_, err = s.session.GetAgent(primitive.NewObjectID())
	s.Error(err)
}

func (s *SessionSuite) TestListAgents() {
	for i := range [5]int{} {
		_, err := s.session.AddAgent(Agent{Name: fmt.Sprintf("aspirin %d", i)})
		s.NoError(err)
	}
	agents, err := s.session.ListAgents(primitive.NilObjectID)
	s.NoError(err)
	s.Equal(5, len(agents))

	for i := range [20]int{} {
		_, err := s.session.AddAgent(Agent{Name: fmt.Sprintf("aspirin %d", i)})
		s.NoError(err)
	}

	agents, err = s.session.ListAgents(primitive.NilObjectID)
	s.NoError(err)
	// reached search limit
	s.Equal(15, len(agents))

	// search with the last agent as offset, it will get the rest of the agents
	agents, err = s.session.ListAgents(agents[len(agents)-1].ID)
	s.NoError(err)
	s.Equal(10, len(agents))
}

func (s *SessionSuite) TestDeleteAgent() {
	for i := range [5]int{} {
		_, err := s.session.AddAgent(Agent{Name: fmt.Sprintf("aspirin %d", i)})
		s.NoError(err)
	}

	agents, err := s.session.ListAgents(primitive.NilObjectID)
	s.NoError(err)
	s.Equal(5, len(agents))

	s.session.DeleteAgent(agents[len(agents)-1].ID)
	agents, err = s.session.ListAgents(primitive.NilObjectID)
	s.NoError(err)
	s.Equal(4, len(agents))

	err = s.session.DeleteAgent(primitive.NewObjectID())
	s.Error(err)
}

func (s *SessionSuite) TestUpdateAgent() {
	id, err := s.session.AddAgent(Agent{Name: "aspirin2d"})
	s.NoError(err)
	err = s.session.UpdateAgent(Agent{ID: id, Name: "aspirin3d"})
	s.NoError(err)
	agent, err := s.session.GetAgent(id)
	s.NoError(err)
	s.Equal("aspirin3d", agent.Name)

	// try to update a not existed agent will cause error
	err = s.session.UpdateAgent(Agent{ID: primitive.NewObjectID(), Name: "aspirin3d"})
	s.Error(err)
}

func TestSessionSuite(t *testing.T) {
	suite.Run(t, new(SessionSuite))
}
