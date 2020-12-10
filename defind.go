package goplugin

import (
	"context"
)

// Hub 插口
type Hub interface {
	Id() interface{}                                                               // 返回id
	Status() uint32                                                                // 插口状态
	Plugin() Plugin                                                                // 返回插件
	Load(Plugin) error                                                             // 加载插件(内存,磁盘,网络)
	UnLoad() error                                                                 // 卸载插件
	Call(ctx context.Context, api interface{}, args ...interface{}) *Ret           // 同步调用 Call(api, 可变参数)
	AsyncCall(ctx context.Context, api interface{}, args ...interface{}) chan *Ret // 异步调用
	Close() error                                                                  // 关闭插口
}

// Plugin 插件
type Plugin interface {
	Id() interface{}                  // 插件id
	Type() string                     // 插件类型
	Version() string                  // 插件版本
	Funcs() map[interface{}]*FuncInfo // 插件函数表 key:api value:func
}

// FuncInfo 函数信息
type FuncInfo struct {
	Fun  interface{} `Fun`  // 函数
	RetN int         `RetN` // 返回值数量
}

// Ret 回调返回
type Ret struct {
	Err     error         `Err`
	HubId   interface{}   `HubId`
	Content []interface{} `Content`
}

// 插件类型
const (
	PTypeMem string = "mem_plugin" // 内存插口
	PTypeDll string = "dll_plugin" // 动态链接库插口
	PTypeNet string = "net_plugin" // 网络插口
	PTypeLua string = "lua_plugin" // lua插口
)

// 插件状态
const (
	PStatLoading   uint32 = 1 // 插件正在插入
	PStatLoaded    uint32 = 2 // 插件已插入
	PStatCalling   uint32 = 3 // 插件正在回调
	PStatReady     uint32 = 4 // 等待回调
	PStatUnloading uint32 = 5 // 插件正在拔出
	PStatUnloaded  uint32 = 6 // 插件已拔出
	PStatClosing   uint32 = 7 // 正在关闭
	PStatClosed    uint32 = 8 // 插件已关闭
	PStatProtect   uint32 = 9 // 插件保护状态
)

// StatusToStr 插件状态转string
func StatusToStr(status uint32) string {
	switch status {
	case PStatLoading:
		return "LOADING"
	case PStatLoaded:
		return "LOADED"
	case PStatCalling:
		return "CALLING"
	case PStatReady:
		return "READY"
	case PStatUnloading:
		return "UNLOADING"
	case PStatUnloaded:
		return "UNLOADED"
	case PStatClosing:
		return "CLOSING"
	case PStatClosed:
		return "CLOSED"
	case PStatProtect:
		return "PROTECT"
	default:
		return "UNKNOWN"
	}
}
