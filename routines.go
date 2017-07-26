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
	value mapset.Set
}

func NewRoutines() *Routines {
	return &Routines{
		value: mapset.NewSet(),
	}
}

func (r *Routines) String() string {
	res := []string{}
	for _, x := range r.value.ToSlice() {
		res = append(res, fmt.Sprintf("%s", x))
	}
	sort.Strings(res)
	return fmt.Sprintf("\"%s\"", strings.Join(res, ","))
}

func (r *Routines) Add(str string) {
	r.value.Add(str)
}

func (r *Routines) Del(str string) {
	r.value.Remove(str)
}
