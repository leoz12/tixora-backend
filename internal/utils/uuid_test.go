package utils_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"tixora/internal/utils"
)

func TestGenerateUUID(t *testing.T) {
	id1 := utils.GenerateUUID()
	id2 := utils.GenerateUUID()

	assert.NotEqual(t, id1, id2)
	_, err := uuid.Parse(id1)
	assert.NoError(t, err, "generated value must be a valid UUID")
}
