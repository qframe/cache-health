package qcache_health

import (
	"sync"
	"github.com/deckarep/golang-set"
	"strings"
	"fmt"
	"sort"
)



type Routines struct {
	mu sync.Mutex
	keys mapset.Set	 `json:"id,omitempty"`
	values map[string]Routine
}

func NewRoutines() *Routines {
	return &Routines{
		keys: mapset.NewSet(),
		values: map[string]Routine{},
	}
}

func (r *Routines) Get() []string {
	res := []string{}
	for _, x := range r.keys.ToSlice() {
		res = append(res, fmt.Sprintf("%s", x))
	}
	sort.Strings(res)
	return res
}

func (r *Routines) Count() int {
	return len(r.Get())
}

func (r *Routines) String() string {
	res := r.Get()
	return fmt.Sprintf("%s", strings.Join(res, ","))
}

func (r *Routines) Add(rt Routine) (err error) {
	key := rt.GetID()
	ok := r.keys.Add(key)
	if ! ok {
		return fmt.Errorf("key '%s' already existing", key)
	}
	r.values[key] = rt
	return
}

func (r *Routines) Del(rt Routine) {
	r.keys.Remove(rt.GetID())
	delete(r.values, rt.GetID())
}
