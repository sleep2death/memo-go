package memo

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/qdrant/go-client/qdrant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	errSessionNotFound = errors.New("Session Not Found")
)

// the agent session
type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	Tags      []string           `bson:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt primitive.DateTime `bson:"created_at,omitempty" json:"created_at,omitempty"` // auto-generated created time
}

type OK struct {
	OK bool `json:"ok" bson:"ok"`
}

type SessionAddResponse struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
}

// @Summary		get all sessions
// @Description	list all sessions
// @Tags			sessions
// @Produce		json
// @Param			offset	query		string	false	"pagination offset id"
// @Param			limit	query		int		false	"pagination limit, default is 5"
// @Success		200		{array}		Session
// @Failure		default		{object}	APIError
// @Router			/s [get]
func (h *Handlers) GetSessions(c *gin.Context) {
	ctx := c.Request.Context()

	offset := c.DefaultQuery("offset", "none")
	limit := c.DefaultQuery("limit", "none")

	filter := bson.M{}

	// set search offset id
	if offset != "none" {
		o, err := primitive.ObjectIDFromHex(offset)
		if err != nil {
			NewError(c, http.StatusBadRequest, err)
			return
		}
		filter["_id"] = bson.M{"$lt": o}
	}

	// set search limit
	l := h.SearchLimit
	if limit != "none" {
		li, err := strconv.Atoi(limit)
		if err != nil {
			NewError(c, http.StatusBadRequest, err)
			return
		}
		l = int64(li)
	}

	opts := options.Find().SetSort(bson.M{"_id": -1}).SetLimit(l)

	cur, err := h.sessions.Find(ctx, filter, opts)
	if err != nil {
		NewError(c, http.StatusNotFound, err)
		return
	}

	var results []Session

	if err = cur.All(ctx, &results); err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	if results == nil {
		results = []Session{}
	}

	c.JSON(http.StatusOK, results)
}

// @Summary		create a session
// @Description	add a sessions
// @Tags			sessions
// @Accept			json
// @Produce		json
// @Param			session	body		Session	true	"the session to be created"
// @Success		200		{object}	mongo.InsertOneResult
// @Failure		default	{object}	APIError
// @Router			/s/add [put]
func (h *Handlers) AddSession(c *gin.Context) {
	ctx := c.Request.Context()

	var p Session
	err := c.ShouldBindJSON(&p)
	if err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	// set create time
	if p.CreatedAt == 0 {
		p.CreatedAt = primitive.NewDateTimeFromTime(time.Now())
	}

	// insert the session
	res, err := h.sessions.InsertOne(
		ctx,
		p,
	)

	if err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	sid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		NewError(c, http.StatusBadRequest, ErrInvalidID)
		return
	}

	// create qdrant collection
	_, err = h.EnsureQCollection(ctx, sid.Hex())
	if err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, SessionAddResponse{ID: sid})
}

// @Summary		get one session by id
// @Description	get one sessions
// @Tags			sessions
// @Accept			json
// @Produce		json
// @Param			id	path		string	true	"the session to get"
// @Success		200	{object}	Session
// @Failure		default	{object}	APIError
// @Router			/s/:id [get]
func (h *Handlers) GetSession(c *gin.Context) {
	ctx := c.Request.Context()

	pid := c.Param("id")
	sid, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		NewError(c, http.StatusBadRequest, err)
		return
	}
	// find the session
	res := h.sessions.FindOne(
		ctx,
		bson.M{"_id": sid},
	)

	err = res.Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			NewError(c, http.StatusNotFound, errSessionNotFound)
			return
		} else {
			NewError(c, http.StatusInternalServerError, err)
			return
		}
	}

	var sess Session
	err = res.Decode(&sess)
	if err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, sess)
}

// @Summary		remove one session
// @Description	remove one sessions
// @Tags			sessions
// @Accept			json
// @Produce		json
// @Param			id	path		string	true	"the session to delete"
// @Success		200	{object}	OK
// @Failure		default	{object}	APIError
// @Router			/s/:id/del [delete]
func (h *Handlers) DeleteSession(c *gin.Context) {
	ctx := c.Request.Context()

	pid := c.Param("id")
	sid, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		NewError(c, http.StatusBadRequest, err)
		return
	}
	// delete the session
	res := h.sessions.FindOneAndDelete(
		ctx,
		bson.M{"_id": sid},
	)
	if err = res.Err(); err != nil {
		// if session not exist, return 404
		if err == mongo.ErrNoDocuments {
			NewError(c, http.StatusNotFound, errSessionNotFound)
			return
		}
		NewError(c, http.StatusBadRequest, err)
		return
	}

	// delete the memories from qdrant
	resp, err := h.qCollections.Delete(ctx, &pb.DeleteCollection{CollectionName: sid.Hex()})
	if err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, OK{OK: resp.Result})
}
