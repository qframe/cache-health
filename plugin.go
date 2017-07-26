package qcache_health

import (
	"context"
	"expvar"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/zpatrick/go-config"
	"github.com/qnib/qframe-types"
	"github.com/qframe/types/health"
	"net"
	"net/http"
	"time"
	"strings"
)

const (
	version   = "0.0.0"
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
}



func New(qChan qtypes.QChan, cfg *config.Config, name string) (Plugin, error) {
	p := qtypes.NewNamedPlugin(qChan, cfg, pluginTyp, pluginPkg, name, version)
	return Plugin{
		Plugin: 			p,
		logRoutines: 		NewRoutines(),
		logSkipRoutines:	NewRoutines(),
		statsRoutines: 		NewRoutines(),
		healthState:		expvar.NewString("health"),
		healthMsg:			expvar.NewString("healthMsg"),
	}, nil
}

func (p *Plugin) PublishExpVars() {
	expvar.Publish("statsRoutines", p.statsRoutines)
	expvar.Publish("logRoutines", p.logRoutines)
	expvar.Publish("logSkipRoutines", p.logSkipRoutines)
	p.healthState.Set("true")
	p.healthMsg.Set("Just started")
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
	time.Sleep(time.Second*time.Duration(2))
	go p.PublishExpVars()
	for {
		select {
		case <-tc.Read:
			p.checkHealth()
		case val := <-dc.Read:
			switch val.(type) {
			case qtypes_health.HealthBeat:
				hb := val.(qtypes_health.HealthBeat)
				p.Log("info", fmt.Sprintf("Received HealthBeat: %v", hb))
				switch hb.Type {
				case "logRoutine":
					switch hb.Action {
					case "start":
						p.logRoutines.Add(hb.Actor)
					case "stop":
						p.logRoutines.Del(hb.Actor)
					}
				case "logSkipRoutine":
					switch hb.Action {
					case "start":
						p.logSkipRoutines.Add(hb.Actor)
					case "stop":
						p.logSkipRoutines.Del(hb.Actor)
					}
				case "statsRoutine":
					switch hb.Action {
					case "start":
						p.statsRoutines.Add(hb.Actor)
					case "stop":
						p.statsRoutines.Del(hb.Actor)
					}
				}
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
func (p *Plugin) checkHealth() {
	msg := []string{}
	// check currently running containers against the Sets held
	info, err := p.cli.Info(ctx)
	if err != nil {
		msg := fmt.Sprintf("Error during Info(): %s", err)
		p.Log("error", msg)
		p.healthState.Set("unhealthy")
		p.healthMsg.Set(msg)
		return
	}
	msg = append(msg, fmt.Sprintf("RunningContainers:%d", info.ContainersRunning))
	statsCount := p.statsRoutines.Count()
	msg = append(msg, fmt.Sprintf("metricsGoRoutines:%d", statsCount))
	if info.ContainersRunning == statsCount {
		p.healthState.Set("healthy")
	} else {
		p.healthState.Set("unhealthy")
		p.healthMsg.Set(strings.Join(msg, " | "))
		return
	}
	logCount := p.logSkipRoutines.Count() + p.logRoutines.Count()
	msg = append(msg, fmt.Sprintf("logsGoRoutine:(%d [logs] + %d [skipped])", p.logRoutines.Count(), p.logSkipRoutines.Count()))
	if info.ContainersRunning == logCount {
		p.healthState.Set("healthy")
	} else {
		p.healthState.Set("unhealthy")
		p.healthMsg.Set(strings.Join(msg, " | "))
		return
	}
	p.healthMsg.Set(strings.Join(msg, " | "))
}

func (p *Plugin) startHTTP() {
	addr := fmt.Sprintf("0.0.0.0:8123")
	sock, err := net.Listen("tcp", addr)
	if err != nil {
		p.Log("error", err.Error())
	}
	p.Log("info", fmt.Sprintf("Start health-endpoint: %s", addr))
	http.Handle("/health", expvar.Handler())
	http.Serve(sock, nil)
}