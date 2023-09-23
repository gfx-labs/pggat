package gat

type ModuleInfo struct {
}

type Module interface {
	GatModule() ModuleInfo
}
