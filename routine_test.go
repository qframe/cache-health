package qcache_health

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestRoutine_GetID(t *testing.T) {
	got := rt1.GetID()
	assert.Equal(t, "id1", got)
}

func TestRoutine_GetStatus(t *testing.T) {
	got := rt1.GetStatus()
	assert.Equal(t, "start", got)
}

func TestRoutine_GetLastUpdateTime(t *testing.T) {
	got := rt1.GetLastUpdateTime()
	assert.Equal(t, time.Unix(1505927762, 0), got)
}

func TestRoutine_GetUptime(t *testing.T) {
	tsCreated := time.Unix(1505927762, 0)
	tsUpdate := time.Unix(1505927762+120, 0)
	rt := NewRoutine("id1", "running", tsUpdate)
	err := rt1.Update(rt)
	assert.NoError(t, err)
	err = rt1.Update(rt2)
	assert.Error(t, err)
	got := rt1.GetUptime()
	assert.Equal(t, tsUpdate.Sub(tsCreated), got)
}
