package qcache_health

import (
	"net/http"
	"fmt"
	"encoding/json"
	"sort"
	"strings"
	"time"
	gring "github.com/zfjagann/golang-ring"
)

const (
	ringCapacity = 3
	Healthy = "healthy"
	Unhealthy = "unhealthy"
)

type HealthEndpoint struct {
	healthRing 		*gring.Ring
	healthMsgRing 	*gring.Ring
	goRoutines 		map[string]*Routines	`json:"routines,omitempty"`
	vitals			map[string]*Vitals		`json:"vitals,omitempty"`

}

func NewHealthEndpoint(routines []string) *HealthEndpoint {
	r := &gring.Ring{}
	r.SetCapacity(ringCapacity)
	msgR := &gring.Ring{}
	msgR.SetCapacity(ringCapacity)
	r.Enqueue("starting")
	msgR.Enqueue("Just started")
	he := &HealthEndpoint{
		healthRing: r,
		healthMsgRing: msgR,
		goRoutines: map[string]*Routines{},
		vitals: map[string]*Vitals{},
	}
	for _, r := range routines {
		he.goRoutines[r] = NewRoutines()
	}
	return he
}

func (he *HealthEndpoint) SetHealth(status, msg string) (err error) {
	v := []string{}
	for _, e := range he.healthRing.Values() {
		v = append(v, fmt.Sprintf("%s", e))
	}
	m := []string{}
	for _, e := range he.healthMsgRing.Values() {
		m = append(m, fmt.Sprintf("%s", e))
	}

	if status != Healthy && len(v) == 1 {
		return fmt.Errorf("Status initialized with '%s'", status)
	}
	if status != Healthy && len(v) == ringCapacity {
		restIsUnhealthy := true
		for _, i := range v[1:] {
			if i != Unhealthy {
				restIsUnhealthy = false
				break
			}
		}
		if v[0] == Healthy && restIsUnhealthy {
			pair := []string{}
			// As we check whether the ring is completly filled
			for i := 0; i < ringCapacity; i++ {
				pair = append(pair, fmt.Sprintf("%s:'%s'", v[i], m[i]))
			}
			return fmt.Errorf("Status becomes unhealthy for a ring-capacity (%d) duration: [%s]", ringCapacity, strings.Join(pair, ","))
		}
	}
	he.healthRing.Enqueue(status)
	he.healthMsgRing.Enqueue(msg)
	return
}

func (he *HealthEndpoint) AddRoutine(routine, id string) (err error) {
	_, ok := he.goRoutines[routine]
	if !ok {
		return fmt.Errorf("Could not find routine '%s'", routine)
	}
	he.goRoutines[routine].Add(id)
	return
}

func (he *HealthEndpoint) DelRoutine(routine, id string) (err error) {
	_, ok := he.goRoutines[routine]
	if !ok {
		return fmt.Errorf("Could not find routine '%s'", routine)
	}
	he.goRoutines[routine].Del(id)
	return
}

func (he *HealthEndpoint) CountRoutine(routine string) int {
	r, ok := he.goRoutines[routine]
	if !ok {
		return -1
	}
	return r.Count()
}

func (he *HealthEndpoint) GetJSON() map[string]interface{} {
	return he.getJSON(time.Now())
}

func (he *HealthEndpoint) getJSON(t time.Time) map[string]interface{} {
	routines :=  map[string]string{}
	for n, r := range he.goRoutines {
		routines[n] = r.String()
	}
	vitals :=  map[string]interface{}{}
	for n, v := range he.vitals {
		vitals[n] = v.getJSON(t)
	}
	hStatus,hMsg := he.CurrentHealth()
	res := map[string]interface{}{
		"status": hStatus,
		"message": hMsg,
		"routines": routines,
		"vitals": vitals,
	}
	return res
}

func (he *HealthEndpoint) CurrentHealth() (s, m string) {
	sL := []string{}
	for _, e := range he.healthRing.Values() {
		sL = append(sL, fmt.Sprintf("%s", e))
	}
	mL := []string{}
	for _, e := range he.healthMsgRing.Values() {
		mL = append(mL, fmt.Sprintf("%s", e))
	}
	return sL[len(sL)-1], mL[len(mL)-1]
}

func (he *HealthEndpoint) GetTXT() string {
	res := []string{}
	hStatus,hMsg := he.CurrentHealth()
	res = append(res, fmt.Sprintf("health:%s | msg:%s", hStatus,hMsg))
	keys := []string{}
	for k, _ := range he.goRoutines {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, n := range keys {
		r := he.goRoutines[n]
		res = append(res, fmt.Sprintf("%-15s: | %-2d | %s", n, r.Count(), r.String()))
	}
	return strings.Join(append(res, ""), "\n")
}

func (he *HealthEndpoint) Handle(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(he.GetJSON())
	} else {
		fmt.Fprint(w, he.GetTXT())
	}
}

/// Vitals
func (he *HealthEndpoint) UpsertVitals(name, state string , t time.Time) {
	if v, ok := he.vitals[name]; !ok {
		 he.vitals[name] = newVitals(t, state)
	} else {
		v.UpdateLast(t, state)
	}
}