package qcache_health

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestVitals_GetJSON(t *testing.T) {
	now := time.Now()
	t1h := now.Add(time.Hour)
	t2h := now.Add(time.Hour*time.Duration(2))
	v := newVitals(now, "initialized")
	assert.Equal(t, "initialized", v.LastState)
	assert.Equal(t, now, v.LastSign)
	exp := map[string]interface{}{
		"status": "initialized",
		"time_updated": now.Format(time.RFC3339Nano),
		"time_ago": "1h0m0s",
	}
	assert.Equal(t, exp, v.getJSON(t1h))
	v.UpdateLast(t1h, "running")
	assert.Equal(t, "running", v.LastState)
	assert.Equal(t, t1h, v.LastSign)
	exp = map[string]interface{}{
		"status": "running",
		"time_updated": t1h.Format(time.RFC3339Nano),
		"time_ago": "1h0m0s",
	}
	assert.Equal(t, exp, v.getJSON(t2h))
}
