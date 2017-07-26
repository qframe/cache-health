package qcache_health

import (
	"expvar"
	"fmt"
	"github.com/zpatrick/go-config"
	"github.com/qnib/qframe-types"
	"github.com/qframe/types/health"
	"net"
	"net/http"
	"time"
)

const (
	version   = "0.0.0"
	pluginTyp = qtypes.CACHE
	pluginPkg = "health"
)

type Plugin struct {
	qtypes.Plugin
	logRoutines		*Routines
	statsRoutines   *Routines
	healthState		*expvar.String
	healthStatus	*expvar.String
}



func New(qChan qtypes.QChan, cfg *config.Config, name string) (Plugin, error) {
	p := qtypes.NewNamedPlugin(qChan, cfg, pluginTyp, pluginPkg, name, version)
	return Plugin{
		Plugin: 		p,
		logRoutines: 	NewRoutines(),
		statsRoutines: 	NewRoutines(),
		healthState:	expvar.NewString("healthy"),
		healthStatus:	expvar.NewString("health"),
	}, nil
}

func (p *Plugin) PublishExpVars() {
	expvar.Publish("statsRoutines", p.statsRoutines)
	expvar.Publish("logRoutines", p.logRoutines)
	p.healthState.Set("true")
	p.healthStatus.Set("Just started")
}
// Run fetches everything from the Data channel and flushes it to stdout
func (p *Plugin) Run() {
	p.Log("notice", fmt.Sprintf("Start plugin v%s", p.Version))
	dc := p.QChan.Data.Join()
	go p.startHTTP()
	time.Sleep(time.Second*time.Duration(2))
	go p.PublishExpVars()
	for {
		select {
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