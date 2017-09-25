package qcache_health

import (
	"time"
	"fmt"
)

type Routine struct {
	id, status string
	created time.Time
	updated time.Time
	lastBeat time.Time
}

func NewRoutine(id, status string, t time.Time) Routine {
	return Routine{id, status, t, t, t}
}

func (r *Routine) GetID() string {
	return r.id
}

func (r *Routine) GetStatus() string {
	return r.status
}

func (r *Routine) Update(rt Routine) (err error){
	if r.id != rt.id {
		return fmt.Errorf("id missmatch (this.id=%s != other.id=%s", r.id, rt.id)
	}
	r.updated = rt.updated
	return
}

func (r *Routine) GetUptime() time.Duration {
	return r.updated.Sub(r.created)
}

func (r *Routine) GetLastUpdateTime() time.Time {
	return r.updated
}

func (r *Routine) GetDurSinceUpdate() time.Duration {
	return time.Now().Sub(r.updated)
}


