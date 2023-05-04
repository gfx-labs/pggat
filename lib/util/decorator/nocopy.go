package decorator

import "sync"

// NoCopy will cause go vet to warn you about copies of the struct.
type NoCopy struct{}

func (T *NoCopy) Lock()   {}
func (T *NoCopy) Unlock() {}

var _ sync.Locker = (*NoCopy)(nil)
