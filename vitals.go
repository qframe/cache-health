package qcache_health

import (
	"time"
)

type Vitals struct {
	LastSign time.Time
	LastState string
}

func NewVitals() *Vitals {
	return newVitals(time.Now(), "initialized")
}

func newVitals(t time.Time, state string) *Vitals {
	return &Vitals{
		LastSign: t,
		LastState: state,
	}
}

func (v *Vitals) UpdateLast(t time.Time, state string) {
	v.LastSign = t
	v.LastState = state
}


func (v *Vitals) GetJSON() map[string]interface{} {
	return v.getJSON(time.Now())
}

func (v *Vitals) getJSON(t time.Time) map[string]interface{} {
	return map[string]interface{}{
		"status": v.LastState,
		"time_updated": v.LastSign.Format(time.RFC3339Nano),
		"time_ago": t.Sub(v.LastSign).String(),
	}
}