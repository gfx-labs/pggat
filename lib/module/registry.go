package module

import "fmt"

type moduleMap struct {
	m map[string]Module
}

func newModuleMap() *moduleMap {
	return &moduleMap{
		m: map[string]Module{},
	}
}

func (m *moduleMap) Register(name string, module Module) error {
	_, ok := m.m[name]
	if ok {
		return fmt.Errorf("module with name already registered: %s", name)
	}
	m.m[name] = module
	return nil
}
