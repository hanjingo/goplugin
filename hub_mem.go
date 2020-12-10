package goplugin

import (
	"context"
	"errors"
	"fmt"

	"reflect"
	"sync/atomic"

	"github.com/hanjingo/gocore"
)

type MemHub struct {
	p       Plugin
	status  uint32
	retChan chan *Ret
}

func NewMemHub(capa int) *MemHub {
	back := &MemHub{
		status:  PStatUnloaded,
		retChan: make(chan *Ret, capa),
	}
	return back
}

func (hub *MemHub) Id() interface{} {
	if hub.p != nil {
		return hub.p.Id()
	}
	return nil
}

func (hub *MemHub) Status() uint32 {
	return atomic.LoadUint32(&hub.status)
}

func (hub *MemHub) Plugin() Plugin {
	return hub.p
}

func (hub *MemHub) Load(new Plugin) error {
	if !atomic.CompareAndSwapUint32(&hub.status, PStatUnloaded, hub.status) {
		return errors.New("not allow to load plugin, status error")
	}
	oldStat := atomic.SwapUint32(&hub.status, PStatLoading)
	defer atomic.CompareAndSwapUint32(&hub.status, PStatLoading, oldStat)
	if hub.p != nil {
		if !gocore.NewVersion(new.Version()).GreaterThan(gocore.NewVersion(hub.p.Version())) {
			return errors.New("can not use older plugin replace newer plugin")
		}
	}
	hub.p = new
	atomic.SwapUint32(&hub.status, PStatLoaded)
	//todo callback
	atomic.SwapUint32(&hub.status, PStatReady)
	return nil
}

func (hub *MemHub) UnLoad() error {
	if !atomic.CompareAndSwapUint32(&hub.status, PStatReady, hub.status) {
		return errors.New("current stat not allowed unload")
	}
	oldStat := atomic.SwapUint32(&hub.status, PStatUnloading)
	defer atomic.CompareAndSwapUint32(&hub.status, PStatUnloading, oldStat)
	hub.p = nil
	// todo upload call
	atomic.SwapUint32(&hub.status, PStatUnloaded)
	return nil
}

func (hub *MemHub) Call(
	ctx context.Context,
	api interface{},
	args ...interface{},
) *Ret {
	if hub.p == nil {
		return &Ret{Err: fmt.Errorf("lua plugin not exist")}
	}
	back := &Ret{HubId: hub.p.Id()}
	if !atomic.CompareAndSwapUint32(&hub.status, PStatReady, hub.status) {
		back.Err = errors.New("current stat not allowed call")
		return back
	}

	back.Content, back.Err = hub.doCall(api, args...)
	return back
}

func (hub *MemHub) AsyncCall(
	ctx context.Context,
	api interface{},
	args ...interface{},
) chan *Ret {
	back := &Ret{HubId: hub.p.Id()}
	back.Content, back.Err = hub.doCall(api, args...)
	hub.retChan <- back
	return hub.retChan
}

func (hub *MemHub) doCall(
	api interface{},
	args ...interface{},
) ([]interface{}, error) {
	atomic.SwapUint32(&hub.status, PStatCalling)
	defer atomic.SwapUint32(&hub.status, PStatReady)

	out := []interface{}{}
	f, ok := hub.p.Funcs()[api]
	if !ok {
		return nil, errors.New("func not exist")
	}
	fv := reflect.ValueOf(f) //函数体
	iv := []reflect.Value{}  //入参
	if args != nil {
		for _, v := range args {
			iv = append(iv, reflect.ValueOf(v))
		}
	}
	tmp := fv.Call(iv)
	for _, v := range tmp {
		out = append(out, v.Interface().([]interface{})...)
	}
	return out, nil
}

func (hub *MemHub) Close() error {
	if !atomic.CompareAndSwapUint32(&hub.status, PStatUnloaded, hub.status) {
		return errors.New("unload plugin before close")
	}
	atomic.SwapUint32(&hub.status, PStatClosing)
	defer atomic.SwapUint32(&hub.status, PStatClosed)
	return nil
}
