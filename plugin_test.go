package qcache_health

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/zpatrick/go-config"
	"github.com/qframe/types/health"
	"github.com/qframe/types/messages"
	"fmt"
	"github.com/qframe/types/qchannel"
)

func TestPlugin_checkHealth(t *testing.T) {
	qchan := qtypes_qchannel.NewQChan()
	cfg := &config.Config{}
	p, err := New(qchan, cfg, "test")
	assert.NoError(t, err, "Should be created smoothly")
	s, m := p.HealthEndpoint.CurrentHealth()
	assert.Equal(t, "starting", s)
	assert.Equal(t, "Just started", m)
	p.checkHealth(0)
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("log"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("logSkip"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("logWrongType"))
	assert.Equal(t, 0, p.HealthEndpoint.CountRoutine("stats"))
	p.RoutineAdd("log", rt1)
	assert.Equal(t, 1, p.HealthEndpoint.CountRoutine("log"))
	p.checkHealth(0)
	s, m = p.HealthEndpoint.CurrentHealth()
	assert.Equal(t, "unhealthy", s)
	assert.Equal(t, "RunningContainers:0 | metricsGoRoutines:0 | logsGoRoutine:(1 [logs] + 0 [skipped] + 0 [non json-file])", m)
	p.checkHealth(1)
	s, m = p.HealthEndpoint.CurrentHealth()
	assert.Equal(t, "unhealthy", s)
	assert.Equal(t, "RunningContainers:1 | metricsGoRoutines:0", m)
	p.RoutineAdd("stats", rt1)
	p.checkHealth(1)
	assert.Equal(t, 1, p.HealthEndpoint.CountRoutine("stats"))
	s, m = p.HealthEndpoint.CurrentHealth()
	assert.Equal(t, "healthy", s)
	assert.Equal(t, "RunningContainers:1 | metricsGoRoutines:1 | logsGoRoutine:(1 [logs] + 0 [skipped] + 0 [non json-file])", m)

}

func TestPlugin_handleHB(t *testing.T) {
	qchan := qtypes_qchannel.NewQChan()
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
