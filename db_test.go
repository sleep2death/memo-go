package memo

import (
	"context"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func loadEnv(t *testing.T) {
	err := godotenv.Load()
	assert.NoError(t, err)
}

func TestSetupMongo(t *testing.T) {
	loadEnv(t)

	m, err := SetupMongo()
	assert.NoError(t, err)

	err = m.Database().Client().Ping(context.TODO(), nil)
	assert.NoError(t, err)
}

func TestSetupQdrant(t *testing.T) {
	loadEnv(t)

	_, _, err := SetupQdrant()
	assert.NoError(t, err)
}
