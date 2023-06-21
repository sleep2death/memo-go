package memo

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// the agent session
type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt primitive.DateTime `bson:"created_at" json:"created_at"`
}

// ShowAccount godoc
//
//	@Summary		get all sessions
//	@Description	list all sessions
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			offset path		string	false	"pagination offset"
//	@Param			limit	path		int		false	"pagination limit"
//	@Success		200		{array}	Session
//	@Failure		400		{object}	HTTPError
//	@Failure		404		{object}	HTTPError
//	@Failure		500		{object}	HTTPError
//	@Router			/s [get]
func (h *Handlers) GetSessions(c *gin.Context) {
	ctx := c.Request.Context()

	filter := bson.M{}
	opts := options.Find().SetSort(bson.M{"_id": -1}).SetLimit(h.SearchLimit)

	cur, err := h.mongo.Collection("sessions").Find(ctx, filter, opts)
	if err != nil {
		NewError(c, http.StatusNotFound, err)
		return
	}

	var results []Session
	if err = cur.All(ctx, &results); err != nil {
		NewError(c, http.StatusInternalServerError, err)
    return
	}

	c.JSON(http.StatusOK, results)
}
