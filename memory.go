package memo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MemoryType int8

const (
	UndefinedMemory MemoryType = iota
	BasicMemory                // default type of memory
	InteractMemory             // agent interacted with someone or something
	PlanMemory                 // the plan which agent is going to follow
)

var (
	MemoryTypeStr = map[uint8]string{
		0: "undefined",
		1: "basic",
		2: "interact",
		3: "plan",
	}
	MemoryTypeInt = map[string]int8{
		"undefined": 0,
		"basic":     1,
		"interact":  2,
		"plan":      3,
	}
)

// String allows MemoryType to implement fmt.Stringer
func (m MemoryType) String() string {
	return MemoryTypeStr[uint8(m)]
}

func (m MemoryType) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *MemoryType) UnmarshalJSON(data []byte) (err error) {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if *m, err = ParseMemoryType(str); err != nil {
		return err
	}
	return nil
}

func ParseMemoryType(s string) (MemoryType, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	value, ok := MemoryTypeInt[s]
	if !ok {
		return MemoryType(0), fmt.Errorf("%q is not a valid memory type", s)
	}
	return MemoryType(value), nil
}

type MemoryMetadata struct {
	Type       MemoryType `bson:"type" json:"type"`
	Content    string     `bson:"content" json:"content"`
	Importance int        `bson:"importance" json:"importance"` // importance score, from 1 to 10
	CreatedAt  time.Time  `bson:"created_at" json:"created_at"`
}

func (m MemoryMetadata) Payload() map[string]*pb.Value {
	return map[string]*pb.Value{
		"content": {Kind: &pb.Value_StringValue{StringValue: m.Content}},
	}
}

type Memory struct {
	ID        string         `bson:"id,omitempty" json:"id,omitempty"`
	Embedding []float32      `bson:"embedding,omitempty" json:"embedding,omitempty"`
	Metadata  MemoryMetadata `bson:"metadata" json:"metadata"`
}

func (m Memory) Point() *pb.PointStruct {
	uuid, _ := uuid.NewUUID() // using uuid v1 which contains timestamp info
	return &pb.PointStruct{
		Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: uuid.String()}},
		Payload: m.Metadata.Payload(),
		Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: m.Embedding}}},
	}
}

type AddMemoriesRequest struct {
	Memories []Memory `bson:"memories" json:"memories"`
}

type AddMemoriesResponse struct {
	IDs []string `bson:"ids" json:"ids"` // inserted memory id in qdrant
}

type SearchMemoryRequest struct {
	Query string `bson:"query" json:"query"`
	Limit int64  `bson:"limit" json:"limit" default:"5"`
}

type RetrieveMemoriesResponse struct {
	Memories []Memory `bson:"memories" json:"memories"` // inserted memory id in qdrant
	Offset   string   `bson:"next_offset" json:"next_offset"`
}

// @Summary		add memories
// @Description	add one or more memories to the session
// @Tags			memories
// @Accept			json
// @Produce		json
// @Param			session	path		string	true	"memory belonging to which session"
// @Param			memory	body		AddMemoryRequest	true	"the memory info"
// @Success		200	{object}	AddMemoryResponse
// @Failure		default	{object}	APIError
// @Router			/m/:session/add [put]
func (hs *Handlers) AddMemories(c *gin.Context) {
	ctx := c.Request.Context()
	sid := c.Param("session") // session id

	// decode body
	var req AddMemoriesRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		NewError(c, http.StatusBadRequest, ErrJSONDecode)
		return
	}

	// build inputs from memories
	var inputs []string = []string{}
	for _, mem := range req.Memories {
		inputs = append(inputs, mem.Metadata.Content)
	}

	// get embeddings from openai api
	em, err := hs.llm.embedding(ctx, inputs)
	if err != nil {
		NewError(c, http.StatusBadRequest, ErrOpenAIEmbedding)
		return
	}

	for _, d := range em.Data {
		req.Memories[d.Index].Embedding = d.Embedding
	}

	ids, err := hs.upsert(ctx, sid, req.Memories)
	if err != nil {
		NewError(c, http.StatusBadRequest, ErrQdrantUpsert)
		return
	}

	c.JSON(http.StatusOK, AddMemoriesResponse{IDs: ids})
}

// @Summary		search memory by similarity
// @Description	search memory by similarity
// @Tags			memories
// @Accept			json
// @Produce		json
// @Param			session	path		string	true	"memory belonging to which session"
// @Param			query	body		SearchMemoryRequest	true	"query object"
// @Success		200	{object}	RetrieveMemoriesResponse
// @Failure		default	{object}	APIError
// @Router			/m/:session/search [get]
func (hs *Handlers) SearchMemories(c *gin.Context) {
	ctx := c.Request.Context()
	sid := c.Param("session") // session id

	// decode body
	var req SearchMemoryRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		NewError(c, http.StatusBadRequest, ErrJSONDecode)
		return
	}

	if req.Limit == 0 {
		req.Limit = hs.SearchLimit
	}

	// get the query's embedding from openai
	embedding, err := hs.llm.embedding(ctx, []string{req.Query})
	if err != nil {
		NewError(c, http.StatusInternalServerError, ErrOpenAIEmbedding)
		return
	}

	// search
	res, err := hs.qPoints.Search(ctx, &pb.SearchPoints{
		CollectionName: sid,
		Vector:         embedding.Data[0].Embedding,
		Limit:          uint64(req.Limit),
		// will not include vectors
		WithVectors: &pb.WithVectorsSelector{SelectorOptions: &pb.WithVectorsSelector_Enable{Enable: false}},
		// will include metadata
		WithPayload: &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		NewError(c, http.StatusInternalServerError, ErrQdrantSearch)
		return
	}

	// covert all points into memories
	var memories []Memory
	for _, r := range res.GetResult() {
		m := Memory{
			ID: r.Id.GetUuid(),
			Metadata: MemoryMetadata{
				Content: r.GetPayload()["content"].GetStringValue(),
			},
		}
		memories = append(memories, m)
	}

	c.JSON(http.StatusOK, RetrieveMemoriesResponse{Memories: memories})
}

// @Summary		get all memories
// @Description	get all memories in this session
// @Tags			memories
// @Accept			json
// @Produce		json
// @Param			session	path		string	true	"memory belonging to which session"
// @Param			offset	query		string	false	"pagination offset id"
// @Param			limit	query		int		false	"pagination limit" default(5)
// @Success		200	{object}	RetrieveMemoriesResponse
// @Failure		default	{object}	APIError
// @Router			/m/:session [get]
func (hs *Handlers) GetAllMemories(c *gin.Context) {
	ctx := c.Request.Context()
	sid := c.Param("session") // session id

	// offset
	offset := c.Query("offset")
	var offsetId *pb.PointId
	if offset != "" {
		offsetId = &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: offset}}
	} else {
		offsetId = nil
	}

	// limit
	limit := c.Query("limit")
	var sLimit uint32
	if limit != "" {
		st, err := strconv.Atoi(limit)
		if err != nil {
			NewError(c, http.StatusBadRequest, err)
			return
		}
		sLimit = uint32(st)
	} else {
		sLimit = uint32(hs.SearchLimit)
	}

	// limit
	resp, err := hs.qPoints.Scroll(ctx, &pb.ScrollPoints{CollectionName: sid, Offset: offsetId, Limit: &sLimit})
	if err != nil {
		log.Println(err)
		NewError(c, http.StatusInternalServerError, ErrQdrantScroll)
		return
	}

	var memories []Memory
	for _, r := range resp.GetResult() {
		m := Memory{
			ID: r.Id.GetUuid(),
			Metadata: MemoryMetadata{
				Content: r.GetPayload()["content"].GetStringValue(),
			},
		}
		memories = append(memories, m)
	}

	c.JSON(http.StatusOK, RetrieveMemoriesResponse{Memories: memories, Offset: resp.GetNextPageOffset().GetUuid()})
}

// ensure qdrant collection MUST exist, if not create one
func (hs *Handlers) EnsureQCollection(ctx context.Context, name string) (upsert bool, err error) {
	_, err = hs.qCollections.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: name})
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
		_, err := hs.qCollections.Create(ctx, &pb.CreateCollection{
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

// upsert memories into qdrant collection
func (hs *Handlers) upsert(ctx context.Context, collection string, memories []Memory) (ids []string, err error) {
	var upsertPoints []*pb.PointStruct

	for _, m := range memories {
		point := m.Point()
		upsertPoints = append(upsertPoints, point)
		ids = append(ids, point.Id.GetUuid())
	}

	waitUpsert := true
	_, err = hs.qPoints.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collection,
		Wait:           &waitUpsert,
		Points:         upsertPoints,
	})

	return
}
