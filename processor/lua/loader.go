package lua

import (
	"context"
	"encoding/json"
	"github.com/skiwer/trident-ci/processor/define"
	"github.com/skiwer/trident-ci/processor/logger"
	lua "github.com/yuin/gopher-lua"
	"gopkg.in/resty.v1"
)

type Loader struct {
	logger     *logger.Logger
	ctx        context.Context
	processCtx *define.ProcessCtx
}

// 检查Table是否为List
func checkList(value lua.LValue) (b bool) {
	if value.Type().String() == "table" {
		b = true
		value.(*lua.LTable).ForEach(func(k, v lua.LValue) {
			if k.Type().String() != "number" {
				b = false
				return
			}
		})
	}
	return
}

func (l *Loader) marshal(data lua.LValue) interface{} {
	switch data.Type() {
	case lua.LTTable:
		if checkList(data) {
			jdata := make([]interface{}, 0)
			data.(*lua.LTable).ForEach(func(key, value lua.LValue) {
				jdata = append(jdata, l.marshal(value))
			})
			return jdata
		} else {
			jdata := map[string]interface{}{}
			data.(*lua.LTable).ForEach(func(key, value lua.LValue) {
				jdata[key.String()] = l.marshal(value)
			})
			return jdata
		}
	case lua.LTNumber:
		return float64(data.(lua.LNumber))
	case lua.LTString:
		return string(data.(lua.LString))
	case lua.LTBool:
		return bool(data.(lua.LBool))
	}
	return nil
}

func (l *Loader) jsonMarshal(L *lua.LState) int {
	data := L.ToTable(1)
	str, err := json.Marshal(l.marshal(data))

	var errMsg string

	if err != nil {
		errMsg = err.Error()
	}

	L.Push(lua.LString(str))
	L.Push(lua.LString(errMsg))

	return 2
}

func (l *Loader) unmarshal(L *lua.LState, data interface{}) lua.LValue {
	switch data.(type) {
	case map[string]interface{}:
		tb := L.NewTable()
		for k, v := range data.(map[string]interface{}) {
			tb.RawSet(lua.LString(k), l.unmarshal(L, v))
		}
		return tb
	case []interface{}:
		tb := L.NewTable()
		for i, v := range data.([]interface{}) {
			tb.Insert(i+1, l.unmarshal(L, v))
		}
		return tb
	case float64:
		return lua.LNumber(data.(float64))
	case string:
		return lua.LString(data.(string))
	case bool:
		return lua.LBool(data.(bool))
	}
	return lua.LNil
}

func (l *Loader) jsonUnMarshal(L *lua.LState) int {
	str := L.ToString(1)
	jdata := map[string]interface{}{}
	err := json.Unmarshal([]byte(str), &jdata)

	var errMsg string

	if err != nil {
		errMsg = err.Error()
	}

	L.Push(l.unmarshal(L, jdata))
	L.Push(lua.LString(errMsg))

	return 2
}

func (l *Loader) getExportFns() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"curlGet":       l.curlGet,
		"curlPost":      l.curlPost,
		"curlPostJson":  l.curlPostJson,
		"log":           l.log,
		"jsonMarshal":   l.jsonMarshal,
		"jsonUnmarshal": l.jsonUnMarshal,
		"getEnv":        l.getEnv,
		"setEnv":        l.setEnv,
		"fail":          l.fail,
	}
}

func (l *Loader) log(L *lua.LState) int {
	str := L.ToString(1)
	l.logger.Info(str)
	return 0
}

func (l *Loader) setEnv(L *lua.LState) int {
	key := L.ToString(1)
	value := L.ToString(2)

	l.processCtx.Env[key] = value

	return 0
}

func (l *Loader) getEnv(L *lua.LState) int {
	key := L.ToString(1)

	L.Push(lua.LString(l.processCtx.Env[key]))

	return 1
}

func (l *Loader) fail(L *lua.LState) int {
	failReason := L.ToString(1)

	l.processCtx.Fail(failReason)

	return 0
}

func (l *Loader) curlGet(L *lua.LState) int {
	url := L.ToString(1)

	client := resty.New()

	resp, err := client.R().SetContext(l.ctx).Get(url)

	var httpCode int
	var httpResp, errorMsg string

	if err != nil {
		errorMsg = err.Error()
	} else {
		httpCode = resp.StatusCode()
		httpResp = string(resp.Body())
	}

	L.Push(lua.LNumber(httpCode))
	L.Push(lua.LString(httpResp))
	L.Push(lua.LString(errorMsg))

	return 3
}

func (l *Loader) curlPost(L *lua.LState) int {
	url := L.ToString(1)
	postData := L.ToString(2)
	contentType := L.ToString(3)

	client := resty.New()

	resp, err := client.R().SetContext(l.ctx).
		SetBody(postData).
		SetHeader("Content-Type", contentType).
		Post(url)

	var httpCode int
	var httpResp, errorMsg string

	if err != nil {
		errorMsg = err.Error()
	} else {
		httpCode = resp.StatusCode()
		httpResp = string(resp.Body())
	}

	L.Push(lua.LNumber(httpCode))
	L.Push(lua.LString(httpResp))
	L.Push(lua.LString(errorMsg))

	return 3
}

func (l *Loader) curlPostJson(L *lua.LState) int {
	url := L.ToString(1)
	postData := L.ToString(2)

	client := resty.New()

	resp, err := client.R().SetContext(l.ctx).
		SetBody(postData).
		SetHeader("Content-Type", "application/json").
		Post(url)

	var httpCode int
	var httpResp, errorMsg string

	if err != nil {
		errorMsg = err.Error()
	} else {
		httpCode = resp.StatusCode()
		httpResp = string(resp.Body())
	}

	L.Push(lua.LNumber(httpCode))
	L.Push(lua.LString(httpResp))
	L.Push(lua.LString(errorMsg))

	return 3
}

func (l *Loader) Load(L *lua.LState) int {
	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), l.getExportFns())

	// returns the module
	L.Push(mod)
	return 1
}
