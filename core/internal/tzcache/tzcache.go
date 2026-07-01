// Package tzcache memoizes time.LoadLocation. The stdlib re-reads tzdata from
// disk on every call; callers that resolve the same handful of zones many
// times per pass (recurrence expansion, iCal timezone resolution) would
// otherwise turn that into a filesystem hot loop.
package tzcache

import (
	"sync"
	"time"
)

var cache sync.Map // string -> *time.Location

// Load is a drop-in, cached replacement for time.LoadLocation.
func Load(name string) (*time.Location, error) {
	if v, ok := cache.Load(name); ok {
		return v.(*time.Location), nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	cache.Store(name, loc)
	return loc, nil
}
