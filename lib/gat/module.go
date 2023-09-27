package gat

type ModuleInfo struct {
	ID  string
	New func() Module
}

type Module interface {
	GatModule() ModuleInfo
}

var modules map[string]ModuleInfo

func RegisterModule(module Module) {
	if modules == nil {
		modules = make(map[string]ModuleInfo)
	}
	info := module.GatModule()
	modules[info.ID] = info
}

func GetModule(id string) (ModuleInfo, bool) {
	info, ok := modules[id]
	return info, ok
}
