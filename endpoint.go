package qcache_health

import (
	"net/http"
	"fmt"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type HealthEndpoint struct {
	health 		string	 				`json:"health,omitempty"`
	healthMsg 	string	 				`json:"healthMsg,omitempty"`
	goRoutines 	map[string]*Routines	`json:"routines,omitempty"`
	vitals		map[string]*Vitals		`json:"vitals,omitempty"`
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

func NewHealthEndpoint(routines []string) *HealthEndpoint {
	he := &HealthEndpoint{
		health: "starting",
		healthMsg: "Just started",
		goRoutines: map[string]*Routines{},
		vitals: map[string]*Vitals{},
	}
	for _, r := range routines {
		he.goRoutines[r] = NewRoutines()
	}
	return he
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

	res := map[string]interface{}{
		"status": he.health,
		"message": he.healthMsg,
		"routines": routines,
		"vitals": vitals,
	}
	return res
}

func (he *HealthEndpoint) GetTXT() string {
	res := []string{}
	res = append(res, fmt.Sprintf("health:%s | msg:%s", he.health, he.healthMsg))
	keys := []string{}
	for k, _ := range he.goRoutines {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, n := range keys {
		r := he.goRoutines[n]
		res = append(res, fmt.Sprintf("%-15s: | %-2d | %s", n, r.Count(), r.String()))
	}
	return strings.Join(res, "\n")
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