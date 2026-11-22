package utils_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"rantau-go/internal/pkg/utils"
)

func TestMap(t *testing.T) {
	got := utils.Map([]int{1, 2}, strconv.Itoa)
	assert.Equal(t, []string{"1", "2"}, got)

	// A nil input must give an empty slice, not nil.
	assert.NotNil(t, utils.Map([]int(nil), strconv.Itoa))
}
