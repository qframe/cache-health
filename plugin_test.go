package qcache_health

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/qnib/qframe-types"
	"github.com/zpatrick/go-config"
	"github.com/qframe/types/health"
	"github.com/qframe/types/messages"
	"fmt"
)

func TestPlugin_checkHealth(t *testing.T) {
	qchan := qtypes.NewQChan()
	cfg := &config.Config{}
	p, err := New(qchan, cfg, "test")
	assert.NoError(t, err, "Should be created smoothly")
	assert.Equal(t, "starting", p.HealthEndpoint.health)
	assert.Equal(t, "Just started", p.HealthEndpoint.healthMsg)
	p.checkHealth(0)
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("log"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("logSkip"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("stats"))
	p.RoutineAdd("log", "id1")
	assert.Equal(t, 1, p.HealthEndpoint.CountRoutine("log"))
	p.checkHealth(0)
	assert.Equal(t, "unhealthy", p.HealthEndpoint.health)
	assert.Equal(t, "RunningContainers:0 | metricsGoRoutines:0 | logsGoRoutine:(1 [logs] + 0 [skipped])", p.HealthEndpoint.healthMsg)
	p.checkHealth(1)
	assert.Equal(t, "unhealthy", p.HealthEndpoint.health)
	assert.Equal(t, "RunningContainers:1 | metricsGoRoutines:0", p.HealthEndpoint.healthMsg)
	p.RoutineAdd("stats", "id1")
	p.checkHealth(1)
	assert.Equal(t, 1, p.HealthEndpoint.CountRoutine("stats"))
	assert.Equal(t, "healthy", p.HealthEndpoint.health)
	assert.Equal(t, "RunningContainers:1 | metricsGoRoutines:1 | logsGoRoutine:(1 [logs] + 0 [skipped])", p.HealthEndpoint.healthMsg)

}

func TestPlugin_handleHB(t *testing.T) {
	qchan := qtypes.NewQChan()
	cfg := &config.Config{}
	p, _ := New(qchan, cfg, "test")
	b := qtypes_messages.NewBase("base")
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("log"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("logSkip"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("stats"))
	for _, r := range []string{"log", "logSkip", "stats"} {
		hb := qtypes_health.NewHealthBeat(b, fmt.Sprintf("routine.%s", r), "id1", "start")
		p.handleHB(hb)
	}
	assert.Equal(t, 1, p.HealthEndpoint.CountRoutine("log"))
	assert.Equal(t, 1, p.HealthEndpoint.CountRoutine("logSkip"))
	assert.Equal(t, 1, p.HealthEndpoint.CountRoutine("stats"))
	for _, r := range []string{"log", "logSkip", "stats"} {
		hb := qtypes_health.NewHealthBeat(b, fmt.Sprintf("routine.%s", r), "id1", "stop")
		p.handleHB(hb)
	}
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("log"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("logSkip"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("stats"))
}