package memo

import (
	"errors"

	"github.com/gin-gonic/gin"
)

var (
	ErrSessionNotFound = errors.New("session not found")
  ErrInvalidID = errors.New("invalid id format")
  ErrJSONDecode = errors.New("can't decode json body")
  ErrOpenAIEmbedding = errors.New("can't create embedding from openai")
  ErrInvalidOpenAPIKey = errors.New("invalid or empty openai api key")
  ErrQdrantUpsert = errors.New("can't upsert points with qdrant")
  ErrQdrantSearch = errors.New("can't search with qdrant")
  ErrQdrantScroll = errors.New("can't scroll points with qdrant")
)

// NewError create a APIError and send it to client
func NewError(ctx *gin.Context, status int, err error) {
	er := APIError{
		Code:    status,
		Message: err.Error(),
	}
	ctx.JSON(status, er)
}

// APIError
type APIError struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"status bad request"`
}
