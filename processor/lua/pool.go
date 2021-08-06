package lua

import (
	"context"
	"sync"
)
import "github.com/yuin/gopher-lua"

type VmPool struct {
	ctx   context.Context
	m     sync.Mutex
	saved []*lua.LState
}

func (pl *VmPool) Get() *lua.LState {
	pl.m.Lock()
	defer pl.m.Unlock()
	n := len(pl.saved)
	if n == 0 {
		return pl.New()
	}
	x := pl.saved[n-1]
	pl.saved = pl.saved[0 : n-1]
	return x
}

func (pl *VmPool) New() *lua.LState {
	L := lua.NewState()
	// setting the L up here.
	// load scripts, set global variables, share channels, etc...

	L.SetContext(pl.ctx)
	return L
}

func (pl *VmPool) Put(L *lua.LState) {
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, L)
}

func (pl *VmPool) Shutdown() {
	for _, L := range pl.saved {
		L.Close()
	}
}

func NewLuaPool(cap int) *VmPool {
	return &VmPool{
		m:     sync.Mutex{},
		saved: make([]*lua.LState, 0, cap),
	}
}
