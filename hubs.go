package goplugin

import (
	"context"
	"errors"
	"sync"

	"github.com/hanjingo/gocore"
)

// Hubs 插件集合
type Hubs struct {
	mu *sync.RWMutex       // mutex
	m  map[interface{}]Hub // hub map
}

// NewHubs
func NewHubs() *Hubs {
	back := &Hubs{
		mu: new(sync.RWMutex),
		m:  make(map[interface{}]Hub),
	}
	return back
}

// LoadPlugin
func (hubs *Hubs) LoadPlugin(p Plugin, args ...interface{}) error {
	hubs.mu.Lock()
	defer hubs.mu.Unlock()

	hub, ok := hubs.m[p.Id()]
	if !ok {
		switch p.Type() {
		case PTypeMem:
			capa := 100
			if len(args) > 0 {
				if arg, ok := args[0].(int); ok {
					capa = arg
				}
			}
			hub = NewMemHub(capa)
		case PTypeLua:
			capa := 100
			if len(args) > 0 {
				if arg, ok := args[0].(int); ok {
					capa = arg
				}
			}
			hub = NewLuaHub(capa)
		default:
			return errors.New("unsupport plugin type")
		}
	}
	if !gocore.NewVersion(p.Version()).GreaterThan(gocore.NewVersion(hub.Plugin().Version())) {
		return errors.New("can not use older plugin replace newer plugin")
	}
	if err := hub.UnLoad(); err != nil {
		return err
	}
	if err := hub.Load(p); err != nil {
		return err
	}
	hubs.m[p.Id()] = hub
	return nil
}

// UnLoadPlugin
func (hubs *Hubs) UnLoadPlugin(id interface{}) error {
	hubs.mu.Lock()
	defer hubs.mu.Unlock()

	if hub, ok := hubs.m[id]; ok {
		if err := hub.UnLoad(); err != nil {
			return err
		}
		delete(hubs.m, id)
	}
	return errors.New("plugin not exist")
}

// Call
func (hubs *Hubs) Call(
	ctx context.Context,
	api interface{},
	args ...interface{},
) []*Ret {
	back := []*Ret{}
	for _, h := range hubs.m {
		if _, ok := h.Plugin().Funcs()[api]; !ok {
			continue
		}
		back = append(back, h.Call(ctx, api, args...))
	}
	return back
}

// AsyncCall
func (hubs *Hubs) AsyncCall(
	api interface{},
	args ...interface{},
) ([]chan *Ret, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	back := []chan *Ret{}
	for _, h := range hubs.m {
		if _, ok := h.Plugin().Funcs()[api]; !ok {
			continue
		}
		back = append(back, h.AsyncCall(ctx, api, args...))
	}
	return back, cancel
}
