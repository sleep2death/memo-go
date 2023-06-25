package memo

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MemoryMetadata struct {
	Content string `bson:"content" json:"content"`
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

type AddMemoryRequest struct {
	Memories []Memory `bson:"memories" json:"memories"`
}

type AddMemoryResponse struct {
	IDs []string `bson:"ids" json:"ids"` // inserted memory id in qdrant
}

type SearchMemoryRequest struct {
	Query string `bson:"query" json:"query"`
	Limit int64  `bson:"limit" json:"limit" default:"5"`
}

type RetrieveMemoriesResponse struct {
	Memories []Memory `bson:"memories" json:"memories"` // inserted memory id in qdrant
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
func (hs *Handlers) AddMemory(c *gin.Context) {
	ctx := c.Request.Context()
	sid := c.Param("session") // session id

	// decode body
	var req AddMemoryRequest
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
	em, err := hs.embedding(ctx, inputs)
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

	c.JSON(http.StatusOK, AddMemoryResponse{IDs: ids})
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
	embedding, err := hs.embedding(ctx, []string{req.Query})
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

	c.JSON(http.StatusOK, RetrieveMemoriesResponse{memories})
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

	c.JSON(http.StatusOK, RetrieveMemoriesResponse{memories})
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

// create embeddings from openai
func (hs *Handlers) embedding(ctx context.Context, input []string) (openai.EmbeddingResponse, error) {
	erq := openai.EmbeddingRequest{
		Input: input,
		Model: openai.AdaEmbeddingV2,
	}
	return hs.openai.CreateEmbeddings(ctx, erq)
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
