package goplugin

import (
	"errors"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/hanjingo/gocore"
	glua "github.com/yuin/gopher-lua"
)

const LuaInfoKey string = "sys_info"

//lua插件信息
type LuaPluginInfo struct {
	Id      interface{}               `Id`
	Type    string                    `Type`
	Version string                    `Version`
	Funcs   map[interface{}]*FuncInfo `Funcs`
}

//lua插件
type LuaPlugin struct {
	lua  *glua.LState
	info *LuaPluginInfo
	src  string
}

func NewLuaPlugin(addr string) *LuaPlugin {
	file, err := os.Open(addr)
	defer file.Close()
	if err != nil {
		return nil
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil
	}

	back := &LuaPlugin{
		lua: glua.NewState(),
		src: string(data),
		info: &LuaPluginInfo{
			Type:  PTypeLua,
			Funcs: make(map[interface{}]*FuncInfo),
		},
	}
	if err := back.init(); err != nil {
		return nil
	}
	return back
}

func (p *LuaPlugin) Id() interface{} {
	return p.info.Id
}

func (p *LuaPlugin) Type() string {
	return p.info.Type
}

func (p *LuaPlugin) Version() string {
	return p.info.Version
}

func (p *LuaPlugin) Funcs() map[interface{}]*FuncInfo {
	return p.info.Funcs
}

//查询lua插件信息
func (p *LuaPlugin) init() error {
	if p.lua == nil {
		return errors.New("插件信息为空")
	}
	if err := p.lua.DoString(p.src); err != nil {
		return err
	}
	back := p.lua.GetGlobal(LuaInfoKey)
	tb, ok := back.(*glua.LTable)
	if !ok {
		return errors.New("lua文件的定义内容不存在")
	}
	info := gocore.Lua2Go(tb, reflect.TypeOf(&LuaPluginInfo{}))
	if info == nil {
		return errors.New("加载lua文件的定义内容失败")
	}
	p.info = info.(*LuaPluginInfo)
	//封装下函数 key:api value:return num
	for k, v := range p.info.Funcs {
		fname := v.Fun.(string)
		nret := v.RetN
		f := func(args ...interface{}) []interface{} {
			in := []glua.LValue{}
			out := []interface{}{}
			for _, v := range args {
				data := gocore.Go2Lua(v)
				in = append(in, data)
			}
			if err := p.lua.CallByParam(glua.P{
				Fn:      p.lua.GetGlobal(fname),
				NRet:    nret,
				Protect: true,
			}, in...); err != nil {
				return out
			}
			for i := 0; i < nret; i++ {
				lv := p.lua.Get(-1)
				p.lua.Pop(1)
				out = append(out, lv)
			}
			return out
		}
		p.info.Funcs[k] = &FuncInfo{Fun: f, RetN: nret}
	}
	return nil
}

//关闭
func (p *LuaPlugin) close() {
	if p.lua != nil {
		p.lua.Close()
	}
}
