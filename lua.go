package servermanager

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	lua "github.com/yuin/gopher-lua"
)

var luaFunctions = make(map[string]lua.LGFunction)

func InitLua(raceControl *RaceControl) {
	luaFunctions["httpRequest"] = LuaHTTPRequest
	luaFunctions["broadcastChat"] = raceControl.LuaBroadcastChat
	luaFunctions["sendChat"] = raceControl.LuaSendChat

	go func() {
		err := managerStartPlugin()

		if err != nil {
			logrus.WithError(err).Error("manager start plugin script failed")
		}
	}()
}

type LuaPlugin struct {
	state *lua.LState

	inputs  []interface{}
	outputs []interface{}
}

func NewLuaPlugin() *LuaPlugin {
	state := lua.NewState()

	for name, fn := range luaFunctions {
		state.SetGlobal(name, state.NewFunction(fn))
	}

	return &LuaPlugin{
		state: state,
	}
}

func (l *LuaPlugin) Inputs(i ...interface{}) *LuaPlugin {
	l.inputs = append(l.inputs, i...)

	return l
}

// remember outputs need to be reversed from whatever the plugin returns
func (l *LuaPlugin) Outputs(o ...interface{}) *LuaPlugin {
	l.outputs = append(l.outputs, o...)

	return l
}

func (l *LuaPlugin) Call(fileName, functionName string) error {
	defer l.state.Close()

	var jsonInputs []lua.LValue

	err := l.state.DoFile(fileName)

	if err != nil {
		return err
	}

	for _, input := range l.inputs {
		jsonInput, err := json.Marshal(input)

		if err != nil {
			return err
		}

		jsonInputs = append(jsonInputs, lua.LString(jsonInput))
	}

	if err := l.state.CallByParam(lua.P{
		Fn:      l.state.GetGlobal(functionName), // name of Lua function
		NRet:    len(l.outputs),                  // number of returned values
		Protect: true,                            // return err or panic
	}, jsonInputs...); err != nil {
		return err
	}

	for i := range l.outputs {
		err := json.Unmarshal([]byte(l.state.Get(-1).String()), l.outputs[i])

		if err != nil {
			return err
		}

		l.state.Pop(1)
	}

	return nil
}

func LuaHTTPRequest(l *lua.LState) int {
	url := l.ToString(1)
	method := l.ToString(2)
	reqBodyString := l.ToString(3)

	httpClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	var req *http.Request
	var err error

	if method != "GET" && reqBodyString != "" {
		req, err = http.NewRequest(method, url, bytes.NewBuffer([]byte(reqBodyString)))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		logrus.WithError(err).Error("Make new request")
		return 0
	}

	req.Header.Set("User-Agent", "assetto-server-manager")

	res, err := httpClient.Do(req)
	if err != nil {
		logrus.WithError(err).Error("do request")
		return 0
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.WithError(err).Error("read response body")
		return 0
	}

	l.Push(lua.LString(body))
	l.Push(lua.LNumber(res.StatusCode))

	return 2
}

func managerStartPlugin() error {
	p := NewLuaPlugin()

	err := p.Call("./plugins/manager.lua", "onManagerStart")

	if err != nil {
		return err
	}

	return nil
}
