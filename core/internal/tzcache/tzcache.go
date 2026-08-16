// Package tzcache memoizes time.LoadLocation. The stdlib re-reads tzdata from
// disk on every call; callers that resolve the same handful of zones many
// times per pass (recurrence expansion, iCal timezone resolution) would
// otherwise turn that into a filesystem hot loop.
package tzcache

import (
	"os"
	"strings"
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

var (
	localOnce sync.Once
	localName string
)

// LocalName resolves the machine's local zone to a loadable IANA name, or ""
// when it cannot be determined. time.Local.String() only carries the name when
// TZ is set; otherwise it is the literal "Local" and the /etc/localtime
// symlink target holds the name.
func LocalName() string {
	localOnce.Do(func() { localName = resolveLocalName() })
	return localName
}

func resolveLocalName() string {
	name := time.Local.String()
	if name == "" || name == "Local" {
		target, err := os.Readlink("/etc/localtime")
		if err != nil {
			return ""
		}
		_, zone, found := strings.Cut(target, "zoneinfo/")
		if !found {
			return ""
		}
		name = zone
	}
	if _, err := Load(name); err != nil {
		return ""
	}
	return name
}
