package memo

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MemoryTestSuite struct {
	suite.Suite
	router *gin.Engine
	hs     *Handlers
	sess   string
}

func (s *MemoryTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	os.Setenv("MONGO_SESSIONS", "test_sessions")

	s.router = gin.New()
	hs, err := Default()
	assert.NoError(s.T(), err)
	s.hs = hs

	s.router.PUT("/s/add", s.hs.AddSession)
	s.router.DELETE("/s/:id/del", s.hs.DeleteSession)

	s.router.PUT("/m/:session/add", s.hs.AddMemory)
	s.router.GET("/m/:session/search", s.hs.SearchMemories)
	s.router.GET("/m/:session", s.hs.GetAllMemories)

	// add a session
	w := httptest.NewRecorder()
	jsonStr := []byte(`{"name":"aspirin2d", "tags":["hello", "world"]}`)
	req, _ := http.NewRequest("PUT", "/s/add", bytes.NewBuffer(jsonStr))
	s.router.ServeHTTP(w, req)
	assert.Equal(s.T(), 200, w.Code)

	var res SessionAddResponse
	err = json.NewDecoder(w.Body).Decode(&res)
	assert.NoError(s.T(), err)
	s.sess = res.ID.Hex()
}

func (s *MemoryTestSuite) TearDownTest() {
	// remove the session
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/s/"+s.sess+"/del", nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(s.T(), 200, w.Code)
}

func (s *MemoryTestSuite) TestAddMemoriesAndSearch() {
	t := s.T()
	w := httptest.NewRecorder()
	jsonStr := []byte(`{"memories":[
    {"metadata":{"content":"hello, my name is aspirin."}}, 
    {"metadata":{"content":"i'm from shanghai."}},
    {"metadata":{"content":"i'm 10 years old."}},
    {"metadata":{"content":"i'm a boy."}},
    {"metadata":{"content":"i'm a little shy"}},
    {"metadata":{"content":"i like playing basketball."}}
  ]}`)
	req, _ := http.NewRequest("PUT", "/m/"+s.sess+"/add", bytes.NewBuffer(jsonStr))
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resp AddMemoryResponse
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, 6, len(resp.IDs))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/m/404/add", nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)

	jsonStr = []byte(`{"query":"where are you from?"}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/m/"+s.sess+"/search", bytes.NewBuffer(jsonStr))
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var result RetrieveMemoriesResponse
	err := json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result.Memories[0].Metadata.Content, "shanghai")
	assert.LessOrEqual(t, int64(len(result.Memories)), s.hs.SearchLimit)

	// set the top k to 3
	jsonStr = []byte(`{"query":"what's your name?", "limit":3}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/m/"+s.sess+"/search", bytes.NewBuffer(jsonStr))
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	err = json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result.Memories[0].Metadata.Content, "aspirin")
	assert.Equal(t, len(result.Memories), 3)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/m/"+s.sess, nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	err = json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, len(result.Memories), 5)
  assert.Contains(t, result.Memories[0].Metadata.Content, "hello")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/m/"+s.sess+"?offset="+result.Offset, nil)
	s.router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	//
	err = json.NewDecoder(w.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, len(result.Memories), 1)
  assert.Contains(t, result.Memories[0].Metadata.Content, "i like")
}

func TestMemory(t *testing.T) {
	suite.Run(t, new(MemoryTestSuite))
}
