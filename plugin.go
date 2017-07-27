package qcache_health

import (
	"context"
	"expvar"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/zpatrick/go-config"
	"github.com/qnib/qframe-types"
	"github.com/qframe/types/health"
	"github.com/urfave/negroni"
	"net/http"
	"time"
	"strings"
)

const (
	version   = "0.1.0"
	pluginTyp = qtypes.CACHE
	pluginPkg = "health"
	dockerAPI = "v1.29"
)

var (
	ctx = context.Background()
)

type Plugin struct {
	qtypes.Plugin
	cli *client.Client
	logRoutines		*Routines
	logSkipRoutines	*Routines
	statsRoutines   *Routines
	healthState		*expvar.String
	healthMsg		*expvar.String
	HealthEndpoint  *HealthEndpoint
}



func New(qChan qtypes.QChan, cfg *config.Config, name string) (Plugin, error) {
	p := qtypes.NewNamedPlugin(qChan, cfg, pluginTyp, pluginPkg, name, version)
	return Plugin{
		Plugin: 			p,
		HealthEndpoint:		NewHealthEndpoint([]string{"log","logSkip", "stats"}),
	}, nil
}


func (p *Plugin) RoutineAdd(routine, id string) {
	err := p.HealthEndpoint.AddRoutine(routine, id)
	if err != nil {
		p.Log("error", err.Error())
	}
}

func (p *Plugin) RoutineDel(routine, id string) {
	err := p.HealthEndpoint.DelRoutine(routine, id)
	if err != nil {
		p.Log("error", err.Error())
	}

}

func (p *Plugin) SetHealth(status, msg string) {
	p.HealthEndpoint.health = status
	p.HealthEndpoint.healthMsg = msg
}

func (p *Plugin) handleHB(hb qtypes_health.HealthBeat) {
	p.Log("trace", fmt.Sprintf("Received HealthBeat: %v", hb))
	switch hb.Type {
	case "routine.log":
		switch hb.Action {
		case "start":
			p.RoutineAdd("log", hb.Actor)
		case "stop":
			p.RoutineDel("log", hb.Actor)
		}
	case "routine.logSkip":
		switch hb.Action {
		case "start":
			p.RoutineAdd("logSkip", hb.Actor)
		case "stop":
			p.RoutineDel("logSkip", hb.Actor)
		}
	case "routine.stats":
		switch hb.Action {
		case "start":
			p.RoutineAdd("stats", hb.Actor)
		case "stop":
			p.RoutineDel("stats", hb.Actor)
		}
	}
}

// Run fetches everything from the Data channel and flushes it to stdout
func (p *Plugin) Run() {
	p.Log("notice", fmt.Sprintf("Start plugin v%s", p.Version))
	dc := p.QChan.Data.Join()
	tc := p.QChan.Tick.Join()
	go p.startHTTP()
	p.StartTicker("health-ticker", 2500)
	err := p.connectingDocker()
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
		}
	}
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
	msg := []string{fmt.Sprintf("RunningContainers:%d", cntCount)}
	statsCnt := p.HealthEndpoint.CountRoutine("stats")
	msg = append(msg, fmt.Sprintf("metricsGoRoutines:%d", statsCnt))
	if cntCount == statsCnt {
		p.SetHealth("healthy", strings.Join(msg, " | "))
	} else {
		p.SetHealth("unhealthy", strings.Join(msg, " | "))
		return
	}
	lCnt := p.HealthEndpoint.CountRoutine("log")
	lSkipCnt := p.HealthEndpoint.CountRoutine("logSkip")
	msg = append(msg, fmt.Sprintf("logsGoRoutine:(%d [logs] + %d [skipped])", lCnt, lSkipCnt))
	if cntCount == (lCnt + lSkipCnt) {
		p.SetHealth("healthy", strings.Join(msg, " | "))
	} else {
		p.SetHealth("unhealthy", strings.Join(msg, " | "))
		return
	}
	p.SetHealth("healthy", strings.Join(msg, " | "))
}

func (p *Plugin) startHTTP() {
	mux := http.NewServeMux()
	mux.HandleFunc("/_health", p.HealthEndpoint.Handle)
	n := negroni.New()
	n.UseHandler(mux)
	n.Use(negroni.HandlerFunc(p.LogMiddleware))
	addr := fmt.Sprintf("0.0.0.0:8123")
	p.Log("info", fmt.Sprintf("Start health-endpoint: %s", addr))
	http.ListenAndServe(addr, n)
}

func (p *Plugin) LogMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	now := time.Now()
	next(rw, r)
	dur := time.Now().Sub(now)
	p.Log("trace", fmt.Sprintf("%s took %s", r.URL.String(), dur.String()))
}