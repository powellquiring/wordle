package bitset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const LENGTH = 2309

func TestBest(t *testing.T) {
	b := New(LENGTH)
	b.SetAll(LENGTH)
	assert.Equal(t, uint(LENGTH), b.Count())
	b = New(LENGTH)
	for bitnum := range LENGTH {
		b.Set(uint(bitnum))
		assert.Equal(t, uint(1+bitnum), b.Count())
	}
}
