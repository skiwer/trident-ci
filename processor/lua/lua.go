package lua

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/processor/define"
	"github.com/skiwer/trident-ci/processor/logger"
)

type Runner struct {
	vmPool *VmPool
}

func NewLuaRunner(pool *VmPool) *Runner {
	return &Runner{
		vmPool: pool,
	}
}

func (r *Runner) wrapLuaScript(script string) string {
	template := `
local m = require("common")
%s
`

	ret := fmt.Sprintf(template, script)

	return ret
}

func (r *Runner) Run(ctx context.Context, workDir string, flowCfg *v1.Flow, processCtx *define.ProcessCtx, logger *logger.Logger) (err error) {
	l := r.vmPool.Get()
	l.SetContext(ctx)

	loader := &Loader{logger: logger, ctx: ctx, processCtx: processCtx}

	l.PreloadModule("common", loader.Load)

	luaScript := flowCfg.LuaCfg.Script

	if !flowCfg.NoEnvRender {
		luaScript = processCtx.RenderByEnv(flowCfg.LuaCfg.Script)
	}

	script := r.wrapLuaScript(luaScript)
	err = l.DoString(script)

	if err != nil {
		return errors.Wrapf(err, "lua脚本执行失败")
	}

	//inputParams := r.getLuaTableFromEnv(l, processCtx.Env)
	//
	//err = l.CallByParam(lua.P{
	//	Fn:      l.GetGlobal("run"),
	//	NRet:    1,
	//	Protect: true,
	//}, inputParams)
	//
	//if err != nil {
	//	return errors.Wrapf(err, "lua脚本执行失败")
	//}
	//
	//ret := l.Get(-1)
	//
	//retTable, ok := ret.(*lua.LTable)
	//
	//if !ok {
	//	return fmt.Errorf("lua脚本返回参数格式错误: %s", ret.String())
	//}
	//
	//retEnvMp := map[string]string{}
	//
	//retTable.ForEach(func(key, value lua.LValue) {
	//	retEnvMp[key.String()] = value.String()
	//})
	//
	//processCtx.AppendEnv(retEnvMp)

	return nil
}
