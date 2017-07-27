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
	value mapset.Set	 `json:"id,omitempty"`
}

func NewRoutines() *Routines {
	return &Routines{
		value: mapset.NewSet(),
	}
}

func (r *Routines) Get() []string {
	res := []string{}
	for _, x := range r.value.ToSlice() {
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

func (r *Routines) Add(str string) {
	r.value.Add(str)
}

func (r *Routines) Del(str string) {
	r.value.Remove(str)
}
