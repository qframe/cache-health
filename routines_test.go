package qcache_health

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestRoutines(t *testing.T) {
	r := NewRoutines()
	r.Add("id1")
	assert.Equal(t, "id1", r.String())
	r.Add("id2")
	assert.Equal(t, "id1,id2", r.String())
	r.Add("id2")
	assert.Equal(t, "id1,id2", r.String())
	r.Del("id1")
	assert.Equal(t, "id2", r.String())
}


