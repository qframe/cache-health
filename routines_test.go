package qcache_health

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestRoutines(t *testing.T) {
	r := NewRoutines()
	ts :=  time.Unix(1505927762, 0)
	rt1 := NewRoutine("id1", "start", ts)
	rt2 := NewRoutine("id2", "start", ts)
	err := r.Add(rt1)
	assert.NoError(t, err, "Should go right in")
	assert.Equal(t, "id1", r.String())
	err = r.Add(rt2)
	assert.NoError(t, err, "Should go right in")
	assert.Equal(t, "id1,id2", r.String())
	err = r.Add(rt2)
	assert.Error(t, err, "Should already be in there")
	assert.Equal(t, "id1,id2", r.String())
	assert.Equal(t, 2, r.Count())
	r.Del(rt1)
	assert.Equal(t, "id2", r.String())
	assert.Equal(t, 1, r.Count())
}


