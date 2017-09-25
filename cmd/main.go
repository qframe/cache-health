package main

import (
	"log"
	"github.com/zpatrick/go-config"
	"github.com/qframe/cache-health"
	"github.com/qframe/collector-docker-events"
	"github.com/qframe/collector-docker-logs"
	"github.com/qframe/types/health"
	"github.com/qframe/types/qchannel"
)

func main() {
	qChan := qtypes_qchannel.NewQChan()
	qChan.Broadcast()
	cfgMap := map[string]string{
		"log.level": "trace",
		"log.only-plugins": "logs",
		"cache.health.ignore-stats": "true",
	}
	cfg := config.NewConfig([]config.Provider{config.NewStatic(cfgMap)})
	// Create Health Cache
	p, err := qcache_health.New(qChan, cfg, "health")
	if err != nil {
		log.Fatalf("[EE] Failed to create cache: %v", err)
	}
	go p.Run()
	// Create docker events collector
	pde, err := qcollector_docker_events.New(qChan, cfg, "events")
	if err != nil {
		log.Fatalf("[EE] Failed to create docker-events: %v", err)
	}
	go pde.Run()
	// Create Docker Logs Collector
	pdl, err := qcollector_docker_logs.New(qChan, cfg, "logs")
	if err != nil {
		log.Fatalf("[EE] Failed to create docker-logs: %v", err)
	}
	go pdl.Run()
	// Create Docker Stats Collector
	/*pds, err := qcollector_docker_stats.New(qChan, cfg, "stats")
	if err != nil {
		log.Fatalf("[EE] Failed to create docker-stats: %v", err)
	}
	go pds.Run()*/
	bg := p.QChan.Data.Join()

	for {
		val := <- bg.Read
		switch val.(type) {
		case qtypes_health.HealthBeat:
			hb := val.(qtypes_health.HealthBeat)
			_ = hb
			//log.Printf("%v", hb)
		}
	}
}
