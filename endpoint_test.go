package qcache_health

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"strings"
	"time"
)

func TestHealthEndpoint(t *testing.T) {
	he := NewHealthEndpoint([]string{"test"})
	assert.Equal(t, -1, he.CountRoutine("nil"))
	assert.Equal(t, 0, he.CountRoutine("test"))
	err := he.AddRoutine("nil", "id1")
	assert.Error(t, err, "Should not find routine 'nil'")
	err = he.AddRoutine("test", "id1")
	assert.NoError(t, err, "Should find routine 'test'")
	assert.Equal(t, 1, he.CountRoutine("test"))
	// remove
	err = he.DelRoutine("nil", "id1")
	assert.Error(t, err, "Should not find routine 'nil'")
	err = he.DelRoutine("test", "id1")
	assert.NoError(t, err, "Should find routine 'test'")
	assert.Equal(t, 0, he.CountRoutine("test"))
}

func TestHealthEndpoint_GetJSONs(t *testing.T) {
	he := NewHealthEndpoint([]string{"test"})
	err := he.AddRoutine("test", "id1")
	assert.NoError(t, err, "Should find routine 'test'")
	err = he.AddRoutine("test", "id2")
	///// Routines
	routines :=  map[string]string{
		"test": "id1,id2",
	}
	///// Vitals
	now := time.Now()
	t1h := now.Add(time.Hour)
	t2h := t1h.Add(time.Hour)
	he.UpsertVitals("v1", "init", now)
	v1Map := map[string]interface{}{
		"status": "init",
		"time_updated": now.Format(time.RFC3339Nano),
		"time_ago": "1h0m0s",
	}
	vitals := map[string]interface{}{
		"v1": v1Map,
	}
	exp := map[string]interface{}{
		"status": "starting",
		"message": "Just started",
		"routines": routines,
		"vitals": vitals,
	}
	got := he.getJSON(t1h)
	// again
	he.UpsertVitals("v1", "running", t1h)
	v1Map = map[string]interface{}{
		"status": "running",
		"time_updated": t1h.Format(time.RFC3339Nano),
		"time_ago": "1h0m0s",
	}
	vitals = map[string]interface{}{
		"v1": v1Map,
	}
	exp = map[string]interface{}{
		"status": "starting",
		"message": "Just started",
		"routines": routines,
		"vitals": vitals,
	}
	got = he.getJSON(t2h)
	assert.Equal(t, exp, got)
}

func TestHealthEndpoint_GetTXT(t *testing.T) {
	he := NewHealthEndpoint([]string{"test"})
	err := he.AddRoutine("test", "id1")
	assert.NoError(t, err, "Should find routine 'test'")
	err = he.AddRoutine("test", "id2")
	exp := []string{
		"health:starting | msg:Just started",
		"test           : | 2  | id1,id2",
	}
	got := he.GetTXT()
	assert.Equal(t, strings.Join(exp,"\n"), got)

}
