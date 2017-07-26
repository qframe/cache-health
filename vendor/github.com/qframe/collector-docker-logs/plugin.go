package qcollector_docker_logs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/zpatrick/go-config"

	"github.com/qnib/qframe-types"
	"regexp"
	"github.com/qframe/types/health"
	"github.com/qframe/types/messages"
	"github.com/qframe/types/docker-events"
)

const (
	version = "0.3.1"
	pluginTyp = "collector"
	pluginPkg = "docker-logs"
	dockerAPI = "v1.29"
)

var (
	ctx = context.Background()
)

type Plugin struct {
	qtypes.Plugin
	cli *client.Client
	info types.Info
	sMap map[string]ContainerSupervisor
	TimeRegex   *regexp.Regexp
}

func (p *Plugin) StartSupervisor(ce events.Message, cnt types.ContainerJSON) {
	s := ContainerSupervisor{
		Plugin: *p,
		Action: ce.Action,
		CntID: ce.Actor.ID,
		CntName: ce.Actor.Attributes["name"],
		Container: cnt,
		Com: make(chan interface{}),
		cli: p.cli,
		qChan: p.QChan,
	}
	s.TimeRegex = regexp.MustCompile(p.CfgStringOr("time-regex", `2\d{3}.*`))
	if p.CfgBoolOr("disable-reparse-logs", false) {
		s.TailRunning = p.CfgStringOr("tail-logs-since", "1m")
	}
	p.sMap[ce.Actor.ID] = s
	go s.Run()
}

func (p *Plugin) StartSupervisorCE(ce qtypes_docker_events.ContainerEvent) {
	p.StartSupervisor(ce.Event, ce.Container)
}


func New(qChan qtypes.QChan, cfg *config.Config, name string) (Plugin, error) {
	var err error
	p := Plugin{
		Plugin: qtypes.NewNamedPlugin(qChan, cfg, pluginTyp, pluginPkg,  name, version),
		sMap: map[string]ContainerSupervisor{},
	}
	return p, err
}

func (p *Plugin) SubscribeRunning() {
	cnts, err := p.cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		p.Log("error", fmt.Sprintf("Failed to list containers: %s", err.Error()))
	} else {
		skipLabel := p.CfgStringOr("skip-container-label", "org.qnib.qframe.skip-log")
		for _, cnt := range cnts {
			cjson, err := p.cli.ContainerInspect(ctx, cnt.ID)
			if err != nil {
				continue
			}
			event := events.Message{
				Type:   "container",
				Action: "running",
				Actor: events.Actor{
					ID: cnt.ID,
					Attributes: map[string]string{"name": strings.Trim(cnt.Names[0],"/")},
				},
			}
			// Skip those with the label:
			skipCnt := false
			for label, _ := range cjson.Config.Labels {
				if label == skipLabel {
					p.Log("info", fmt.Sprintf("Skip subscribing to logs of '%s' as label '%s' is set", cnt.Names, skipLabel))
					b := qtypes_messages.NewTimedBase(p.Name, time.Unix(cnt.Created, 0))
					de := qtypes_docker_events.NewDockerEvent(b, event)
					ce := qtypes_docker_events.NewContainerEvent(de, cjson)
					h := qtypes_health.NewHealthBeat(b, "logSkipRoutine", ce.Container.ID, "start")
					p.Log("info", "Send logSkip-HealthBeat for "+h.Actor)
					p.QChan.SendData(h)
					skipCnt = true
					break
				}
			}
			if skipCnt {
				continue
			}

			b := qtypes_messages.NewTimedBase(p.Name, time.Unix(cnt.Created, 0))
			de := qtypes_docker_events.NewDockerEvent(b, event)
			ce := qtypes_docker_events.NewContainerEvent(de, cjson)
			h := qtypes_health.NewHealthBeat(b, "logRoutine", ce.Container.ID, "start")
			p.QChan.SendData(h)
			p.StartSupervisorCE(ce)
		}
	}
}

func (p *Plugin) Run() {
	p.Log("notice", fmt.Sprintf("Start v%s", p.Version))

	var err error
	dockerHost := p.CfgStringOr("docker-host", "unix:///var/run/docker.sock")
	p.cli, err = client.NewClient(dockerHost, dockerAPI, nil, nil)
	if err != nil {
		p.Log("error", fmt.Sprintf("Could not connect docker/docker/client to '%s': %v", dockerHost, err))
		return
	}
	p.info, err = p.cli.Info(ctx)
	if err != nil {
		p.Log("error", fmt.Sprintf("Error during Info(): %v >err> %s", p.info, err))
		return
	} else {
		p.Log("info", fmt.Sprintf("Connected to '%s' / v'%s' (SWARM: %s)", p.info.Name, p.info.ServerVersion, p.info.Swarm.LocalNodeState))
	}
	// need to start listener for all containers
	skipRunning := p.CfgBoolOr("skip-running", false)
	if ! skipRunning {
		p.Log("info", fmt.Sprintf("Start listeners for already running containers: %d", p.info.ContainersRunning))
		p.SubscribeRunning()
	}
	inputs := p.GetInputs()
	srcSuccess := p.CfgBoolOr("source-success", true)
	dc := p.QChan.Data.Join()
	for {
		select {
		case msg := <-dc.Read:
			switch msg.(type) {
			case qtypes_docker_events.ContainerEvent:
				ce := msg.(qtypes_docker_events.ContainerEvent)
				if len(inputs) != 0 && ! ce.InputsMatch(inputs) {
					continue
				}
				if ce.SourceSuccess != srcSuccess {
					continue
				}
				if ce.Event.Type == "container" && (strings.HasPrefix(ce.Event.Action, "exec_create") || strings.HasPrefix(ce.Event.Action, "exec_start")) {
					continue
				}
				p.Log("debug", fmt.Sprintf("Received: %s", ce.Message))
				switch ce.Event.Type {
				case "container":
					switch ce.Event.Action {
					case "start":
						p.sendHealthhbeat(ce, "start")
						p.StartSupervisorCE(ce)
					case "die":
						p.sendHealthhbeat(ce, "stop")
						p.sMap[ce.Event.Actor.ID].Com <- ce.Event.Action
					}
				}
			}
		}
	}
}


func (p *Plugin) sendHealthhbeat(ce qtypes_docker_events.ContainerEvent, action string) {
	skipLabel := p.CfgStringOr("skip-container-label", "org.qnib.qframe.skip-log")
	b := qtypes_messages.NewTimedBase(p.Name, ce.Time)
	// Skip those with the label:
	routineName := "logRoutine"
	for label, _ := range ce.Container.Config.Labels {
		if label == skipLabel {
			routineName = "logSkipRoutine"
			break
		}
	}
	h := qtypes_health.NewHealthBeat(b, routineName, ce.Container.ID, action)
	p.QChan.SendData(h)
}