package qcache_health

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/qframe/types/health"
	"github.com/urfave/negroni"
	"net/http"
	"time"
	"strings"
	"github.com/qframe/types/constants"
	"github.com/qframe/types/plugin"
	"github.com/qframe/types/qchannel"
	"github.com/zpatrick/go-config"
)

const (
	version   = "0.1.3"
	pluginTyp = qtypes_constants.CACHE
	pluginPkg = "health"
	dockerAPI = "v1.29"
)

var (
	ctx = context.Background()
)

type Plugin struct {
	*qtypes_plugin.Plugin
	cli *client.Client
	HealthEndpoint  *HealthEndpoint
}



func New(qChan qtypes_qchannel.QChan, cfg *config.Config, name string) (Plugin, error) {
	p := qtypes_plugin.NewNamedPlugin(qChan, cfg, pluginTyp, pluginPkg, name, version)
	ignoreStats := p.CfgBoolOr("ignore-stats", false)
	ignoreLogs := p.CfgBoolOr("ignore-logs", false)
	he := NewHealthEndpoint([]string{"log","logSkip", "logWrongType", "stats"})
	if ignoreLogs {
		he = NewHealthEndpoint([]string{"stats"})
	}
	if ignoreStats {
		he = NewHealthEndpoint([]string{"log","logSkip", "logWrongType"})
	}
	return Plugin{
		Plugin: p,
		HealthEndpoint:	he,
	}, nil
}

func (p *Plugin) RoutineAdd(routineType string, rt Routine) {
	err := p.HealthEndpoint.AddRoutine(routineType, rt)
	if err != nil {
		p.Log("error", err.Error())
	}
}

func (p *Plugin) RoutineDel(routineType string, rt Routine) {
	err := p.HealthEndpoint.DelRoutine(routineType, rt)
	if err != nil {
		p.Log("error", err.Error())
	}
}

func (p *Plugin) SetHealth(status, msg string) {
	err := p.HealthEndpoint.SetHealth(status, msg)
	if err != nil {
		p.Log("error", fmt.Sprintf("%s for msg '%s': %s", status, msg, err.Error()))
	}
}

func (p *Plugin) handleRoutines(hb qtypes_health.HealthBeat) {
	rt := NewRoutine(hb.Actor, hb.Action, hb.Time)
	switch hb.Type {
	case "routine.log":
		switch hb.Action {
		case "start":
			p.RoutineAdd("log", rt)
		case "stop":
			p.RoutineDel("log", rt)
		}
	case "routine.logSkip":
		switch hb.Action {
		case "start":
			p.RoutineAdd("logSkip", rt)
		case "stop":
			p.RoutineDel("logSkip", rt)
		}
	case "routine.logWrongType":
		switch hb.Action {
		case "start":
			p.RoutineAdd("logWrongType", rt)
		case "stop":
			p.RoutineDel("logWrongType", rt)
		}
	case "routine.stats":
		switch hb.Action {
		case "start":
			p.RoutineAdd("stats", rt)
		case "stop":
			p.RoutineDel("stats", rt)
		}
	}
}

func (p *Plugin) handleVitals(hb qtypes_health.HealthBeat) {
	p.HealthEndpoint.UpsertVitals(hb.Actor, hb.Action, hb.Time)
}

func (p *Plugin) handleHB(hb qtypes_health.HealthBeat) {
	p.Log("debug", fmt.Sprintf("Received HealthBeat: %v", hb))
	switch {
	case strings.HasPrefix(hb.Type, "routine."):
		p.handleRoutines(hb)
	case hb.Type == "vitals":
		p.handleVitals(hb)
	}
}

// Run fetches everything from the Data channel and flushes it to stdout
func (p *Plugin) Run() (err error) {
	p.Log("notice", fmt.Sprintf("Start plugin v%s", p.Version))
	dc := p.QChan.Data.Join()
	done := p.QChan.Done.Join()
	tc := p.QChan.Tick.Join()
	go p.startHTTP()
	p.StartTicker("health-ticker", 2500)
	err = p.connectingDocker()
	if err != nil {
		return
	}
	for {
		select {
		case <-tc.Read:
			cntCount := p.getRunningCntCount()
			p.checkHealth(cntCount)
		case val := <-dc.Read:
			switch val.(type) {
			case qtypes_health.HealthBeat:
				hb := val.(qtypes_health.HealthBeat)
				p.handleHB(hb)
			}
		case err = <- p.ErrChan:
			return
		case <- done.Read:
			return
		}
	}
	return
}

func (p *Plugin) connectingDocker() (err error) {
	dockerHost := p.CfgStringOr("docker-host", "unix:///var/run/docker.sock")
	p.cli, err = client.NewClient(dockerHost, dockerAPI, nil, nil)
	if err != nil {
		p.Log("error", fmt.Sprintf("Could not connect docker/docker/client to '%s': %v", dockerHost, err))
	}
	return
}

func (p *Plugin) getRunningCntCount() int {
	info, err := p.cli.Info(ctx)
	if err != nil {
		msg := fmt.Sprintf("Error during Info(): %s", err)
		p.Log("error", msg)
		p.SetHealth("unhealthy", msg)
		return -1
	}
	return info.ContainersRunning
}

func (p *Plugin) checkHealth(cntCount int) {
	ignoreStats := p.CfgBoolOr("ignore-stats", false)
	ignoreLogs := p.CfgBoolOr("ignore-logs", false)
	msg := []string{fmt.Sprintf("RunningContainers:%d", cntCount)}
	if ! ignoreStats {
		statsCnt := p.HealthEndpoint.CountRoutine("stats")
		msg = append(msg, fmt.Sprintf("metricsGoRoutines:%d", statsCnt))
		if cntCount == statsCnt {
			p.SetHealth("healthy", strings.Join(msg, " | "))
		} else {
			p.SetHealth("unhealthy", strings.Join(msg, " | "))
			return
		}
	}
	if !ignoreLogs {
		lCnt := p.HealthEndpoint.CountRoutine("log")
		lSkipCnt := p.HealthEndpoint.CountRoutine("logSkip")
		lWrongType := p.HealthEndpoint.CountRoutine("logWrongType")
		msg = append(msg, fmt.Sprintf("logsGoRoutine:(%d [logs] + %d [skipped] + %d [non json-file])", lCnt, lSkipCnt, lWrongType))
		if cntCount == (lCnt + lSkipCnt + lWrongType) {
			p.SetHealth("healthy", strings.Join(msg, " | "))
		} else {
			p.SetHealth("unhealthy", strings.Join(msg, " | "))
			return
		}
	}
	p.SetHealth("healthy", strings.Join(msg, " | "))
}

func (p *Plugin) startHTTP() {
	bindHost := p.CfgStringOr("bind-host", "0.0.0.0")
	bindPort := p.CfgStringOr("bind-port", "8123")
	bindAddr := fmt.Sprintf("%s:%s", bindHost, bindPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/_health", p.HealthEndpoint.Handle)
	n := negroni.New()
	n.UseHandler(mux)
	n.Use(negroni.HandlerFunc(p.LogMiddleware))
	p.Log("info", fmt.Sprintf("Start health-endpoint: %s", bindAddr))
	err :=  http.ListenAndServe(bindAddr, n)
	p.ErrChan <- err
	p.Log("error", err.Error())
}

func (p *Plugin) LogMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	now := time.Now()
	next(rw, r)
	dur := time.Now().Sub(now)
	p.Log("trace", fmt.Sprintf("%s took %s", r.URL.String(), dur.String()))
}
