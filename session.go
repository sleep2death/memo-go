package memo

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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

// ShowAccount godoc
//
//	@Summary		get all sessions
//	@Description	list all sessions
//	@Tags			sessions
//	@Produce		json
//	@Param			offset	query		string	false	"pagination offset"
//	@Param			limit	query		int		false	"pagination limit, default is 5"
//	@Success		200		{array}		Session
//	@Failure		400		{object}	HTTPError
//	@Failure		404		{object}	HTTPError
//	@Failure		500		{object}	HTTPError
//	@Router			/s [get]
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

	cur, err := h.mongo.Collection(os.Getenv("MONGO_SESSIONS")).Find(ctx, filter, opts)
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

// ShowAccount godoc
//
//	@Summary		create a session
//	@Description	add a sessions
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			session	body		Session	true	"the session to be created"
//	@Success		200		{object}	mongo.InsertOneResult
//	@Failure		400		{object}	HTTPError
//	@Failure		404		{object}	HTTPError
//	@Failure		500		{object}	HTTPError
//	@Router			/s/add [put]
func (h *Handlers) AddSession(c *gin.Context) {
	ctx := c.Request.Context()

	var p Session
	err := c.ShouldBindJSON(&p)
	if err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	// set create time
	p.CreatedAt = primitive.NewDateTimeFromTime(time.Now())

	// insert the session
	res, err := h.mongo.Collection(os.Getenv("MONGO_SESSIONS")).InsertOne(
		ctx,
		p,
	)

	if err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// ShowAccount godoc
//
//	@Summary		get one session by id
//	@Description	get one sessions
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"the session to get"
//	@Success		200	{object}	Session
//	@Failure		400	{object}	HTTPError
//	@Failure		404	{object}	HTTPError
//	@Failure		500	{object}	HTTPError
//	@Router			/s/:id [get]
func (h *Handlers) GetSession(c *gin.Context) {
	ctx := c.Request.Context()

	pid := c.Param("id")
	sid, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		NewError(c, http.StatusBadRequest, err)
		return
	}
	// find the session
	res := h.mongo.Collection(os.Getenv("MONGO_SESSIONS")).FindOne(
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

// ShowAccount godoc
//
//	@Summary		remove one session
//	@Description	remove one sessions
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"the session to delete"
//	@Success		200	{object}	mongo.DeleteResult
//	@Failure		400	{object}	HTTPError
//	@Failure		404	{object}	HTTPError
//	@Failure		500	{object}	HTTPError
//	@Router			/s/:id/del [delete]
func (h *Handlers) DeleteSession(c *gin.Context) {
	ctx := c.Request.Context()

	pid := c.Param("id")
	sid, err := primitive.ObjectIDFromHex(pid)
	if err != nil {
		NewError(c, http.StatusBadRequest, err)
		return
	}
	// delete the session
	res, err := h.mongo.Collection(os.Getenv("MONGO_SESSIONS")).DeleteOne(
		ctx,
		bson.M{"_id": sid},
	)

	if err != nil {
		NewError(c, http.StatusInternalServerError, err)
		return
	}

	if res.DeletedCount == 0 {
		NewError(c, http.StatusBadRequest, errSessionNotFound)
		return
	}

	c.JSON(http.StatusOK, res)
}
