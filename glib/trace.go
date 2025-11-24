package glib

import (
	"fmt"
	"runtime"
	"runtime/pprof"
	"unsafe"
	"sync"
	"time"
)

var gObjectProfile *pprof.Profile
var closureProfile *pprof.Profile
var glibGCProfile *pprof.Profile

var aliveObjects = make(map[string]int)
var aliveObjectsMu sync.RWMutex

func init() {
	objects := "go-glib-reffed-objects"
	gObjectProfile = pprof.Lookup(objects)
	if gObjectProfile == nil {
		gObjectProfile = pprof.NewProfile(objects)
	}

	closures := "go-glib-active-closures"
	closureProfile = pprof.Lookup(closures)
	if closureProfile == nil {
		closureProfile = pprof.NewProfile(closures)
	}

	gc := "go-glib-gc-calls"
	glibGCProfile = pprof.Lookup(gc)
	if glibGCProfile == nil {
		glibGCProfile = pprof.NewProfile(gc)
	}

	go func() {
		for {
			<-time.After(1 * time.Second)
			aliveObjectsMu.RLock()
			fmt.Printf("GLIB: alive objects:\n")
			for name, count := range aliveObjects {
				fmt.Printf("GLIB:   %s: %d\n", name, count)
			}
			aliveObjectsMu.RUnlock()
		}
	}()
}


func WrapFinalizer[T any](name string, obj *T, finalizer func(*T)) {
	aliveObjectsMu.Lock()
	aliveObjects[name]++
	glibGCProfile.Add(uintptr(unsafe.Pointer(obj)), 1)
	aliveObjectsMu.Unlock()
	//fmt.Printf("FINALIZER[%s]: setting up\n", name)
	runtime.SetFinalizer(obj, func(o *T) {
		aliveObjectsMu.Lock()
		aliveObjects[name]--
		glibGCProfile.Remove(uintptr(unsafe.Pointer(o)))
		aliveObjectsMu.Unlock()
		//fmt.Printf("FINALIZER[%s]: running, alive objects: %d\n", name, aliveObjects[name])
		finalizer(o)
	})
}
