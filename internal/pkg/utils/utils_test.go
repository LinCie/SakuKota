package utils_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"rantau-go/internal/pkg/utils"
)

func TestMap(t *testing.T) {
	assert.Equal(t, []string{"1", "2"}, utils.Map([]int{1, 2}, strconv.Itoa))
	assert.NotNil(t, utils.Map([]int(nil), strconv.Itoa), "nil input must give an empty slice, not nil")
}
