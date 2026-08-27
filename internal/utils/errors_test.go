package utils_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"tixora/internal/utils"
)

func TestAppError_ErrorMessage(t *testing.T) {
	wrapped := errors.New("underlying cause")

	withErr := utils.NewAppError(400, "bad request", wrapped)
	assert.Equal(t, "bad request: underlying cause", withErr.Error())

	withoutErr := utils.NewAppError(404, "not found", nil)
	assert.Equal(t, "not found", withoutErr.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	wrapped := errors.New("underlying cause")
	appErr := utils.NewAppError(400, "bad request", wrapped)

	assert.ErrorIs(t, appErr, wrapped)
	assert.Equal(t, wrapped, errors.Unwrap(appErr))
}

func TestAppError_AsWorksThroughFmtErrorf(t *testing.T) {
	appErr := utils.NewAppError(409, "conflict", utils.ErrConflict)

	var target *utils.AppError
	assert.True(t, errors.As(appErr, &target))
	assert.Equal(t, 409, target.Code)
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []error{
		utils.ErrNotFound,
		utils.ErrInvalidInput,
		utils.ErrUnauthorized,
		utils.ErrForbidden,
		utils.ErrConflict,
	}

	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			assert.False(t, errors.Is(a, b), "sentinel %v must not match sentinel %v", a, b)
		}
	}
}
