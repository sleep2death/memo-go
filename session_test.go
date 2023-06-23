package memo

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SessionTestSuite struct {
	suite.Suite
	router *gin.Engine
	hs     *Handlers
}

func (s *SessionTestSuite) SetupSuite() {
	gin.SetMode(gin.ReleaseMode)
	os.Setenv("MONGO_SESSIONS", "test_sessions")

	s.router = gin.Default()
	hs, err := New()
	assert.NoError(s.T(), err)
	s.hs = hs

	s.router.GET("/s", s.hs.GetSessions)
	s.router.PUT("/s/add", s.hs.AddSession)
	s.router.DELETE("/s/:id/del", s.hs.DeleteSession)
	s.router.GET("/s/:id", s.hs.GetSession)
}

func (s *SessionTestSuite) TearDownTest() {
	// clear testing data here
	err := s.hs.mongo.Collection("test_sessions").Drop(context.TODO())
	assert.NoError(s.T(), err)
}

func (s *SessionTestSuite) TestSession() {
	t := s.T()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/s", nil)
	s.router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var p []Session
	json.NewDecoder(w.Body).Decode(&p)
	assert.Zero(t, len(p))

	// add 15 sessions
	for i := 0; i < 15; i++ {
		w := httptest.NewRecorder()
		jsonStr := []byte(`{"name":"aspirin2d", "tags":["hello", "world"]}`)
		req, _ = http.NewRequest("PUT", "/s/add", bytes.NewBuffer(jsonStr))
		s.router.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}

	// first page
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/s?limit=6", nil)
	s.router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	json.NewDecoder(w.Body).Decode(&p)
	assert.Equal(t, 6, len(p))

	// page Two
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/s?limit=6&offset="+p[len(p)-1].ID.Hex(), nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	json.NewDecoder(w.Body).Decode(&p)
	assert.Equal(t, 6, len(p))

	// last page
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/s?limit=6&offset="+p[len(p)-1].ID.Hex(), nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	json.NewDecoder(w.Body).Decode(&p)
	assert.Equal(t, 3, len(p))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/s?offset=123", nil)
	s.router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/s?limit=0.3", nil)
	s.router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/s/"+p[len(p)-1].ID.Hex()+"/del", nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/s/123/del", nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/s/"+primitive.NewObjectID().Hex()+"/del", nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/s?limit=20", nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	json.NewDecoder(w.Body).Decode(&p)
	assert.Equal(t, 14, len(p))

	w = httptest.NewRecorder()
	jsonStr := []byte(`{"name":"aspirin2ds", "tags":["hello", "world!"]}`)
	req, _ = http.NewRequest("PUT", "/s/add", bytes.NewBuffer(jsonStr))
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var ir mongo.InsertOneResult
	err := json.NewDecoder(w.Body).Decode(&ir)
	assert.NoError(t, err)

	objID, ok := ir.InsertedID.(string)
	assert.True(t, ok)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/s/"+objID, nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var sess Session
	json.NewDecoder(w.Body).Decode(&sess)
	assert.Equal(t, "world!", sess.Tags[1])
}

func TestSessionTestSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}
