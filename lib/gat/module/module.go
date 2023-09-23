package module

type ModuleInfo struct {
	ID  string
	New func() Module
}

type Module interface {
	GatModule() ModuleInfo
}

var globalModuleMap = newModuleMap()

func Register(name string, module Module) {
	err := globalModuleMap.Register(name, module)
	if err != nil {
		panic(err)
	}
}
