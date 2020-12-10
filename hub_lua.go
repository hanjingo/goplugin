package goplugin

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
)

type LuaHub struct {
	p      *LuaPlugin
	status uint32
	out    chan *Ret
}

func NewLuaHub(capa int) *LuaHub {
	back := &LuaHub{
		status: PStatUnloaded,
		out:    make(chan *Ret, capa),
	}
	return back
}

func (hub *LuaHub) Id() interface{} {
	if hub.p != nil {
		return hub.p.Id()
	}
	return nil
}

func (hub *LuaHub) Status() uint32 {
	return hub.status
}

func (hub *LuaHub) Plugin() Plugin {
	return hub.p
}

func (hub *LuaHub) Load(p Plugin) error {
	if !atomic.CompareAndSwapUint32(&hub.status, PStatUnloaded, hub.status) {
		return fmt.Errorf("load with status error")
	}
	atomic.SwapUint32(&hub.status, PStatLoading)
	defer atomic.SwapUint32(&hub.status, PStatLoaded)

	lplugin, ok := p.(*LuaPlugin)
	if !ok {
		return fmt.Errorf("only support lua plugin")
	}
	hub.p = lplugin
	if err := hub.p.init(); err != nil {
		hub.doUnLoad()
		return err
	}
	return nil
}

func (hub *LuaHub) UnLoad() error {
	if !atomic.CompareAndSwapUint32(&hub.status, PStatReady, hub.status) {
		return fmt.Errorf("unload with status error")
	}
	return hub.doUnLoad()
}
func (hub *LuaHub) doUnLoad() error {
	atomic.SwapUint32(&hub.status, PStatUnloading)
	defer atomic.SwapUint32(&hub.status, PStatUnloaded)

	hub.p.close()
	hub.p = nil
	return nil
}

func (hub *LuaHub) Call(
	ctx context.Context,
	api interface{},
	args ...interface{},
) *Ret {
	if hub.p == nil {
		return &Ret{Err: fmt.Errorf("lua plugin not exist")}
	}
	back := &Ret{HubId: hub.p.Id()}
	if !atomic.CompareAndSwapUint32(&hub.status, PStatReady, hub.status) && !atomic.CompareAndSwapUint32(&hub.status, PStatCalling, hub.status) {
		back.Err = fmt.Errorf("plugin invalid")
		return back
	}
	back.Content, back.Err = hub.doCall(api, args...)
	return back
}

func (hub *LuaHub) AsyncCall(
	ctx context.Context,
	api interface{},
	args ...interface{},
) chan *Ret {
	if hub.p == nil {
		hub.out <- &Ret{Err: fmt.Errorf("lua plugin not exist")}
		return hub.out
	}
	back := &Ret{HubId: hub.p.Id()}
	back.Content, back.Err = hub.doCall(api, args...)
	hub.out <- back
	return hub.out
}

func (hub *LuaHub) doCall(
	api interface{},
	args ...interface{},
) ([]interface{}, error) {
	atomic.SwapUint32(&hub.status, PStatCalling)
	defer atomic.SwapUint32(&hub.status, PStatReady)

	out := []interface{}{}
	fun, ok := hub.p.Funcs()[api]
	if !ok || fun == nil {
		return out, fmt.Errorf("api:%s not exist", api)
	}
	fv := reflect.ValueOf(fun) //函数体
	iv := []reflect.Value{}    //入参
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

//关闭
func (hub *LuaHub) Close() error {
	if !atomic.CompareAndSwapUint32(&hub.status, PStatUnloaded, hub.status) {
		return fmt.Errorf("unload plugin first")
	}
	return hub.doClose()
}
func (hub *LuaHub) doClose() error {
	atomic.SwapUint32(&hub.status, PStatClosing)
	hub.p.close()
	atomic.SwapUint32(&hub.status, PStatClosed)
	return nil
}
