package fed

// idleChecker is implemented by codecs that can inspect the transport without blocking.
type idleChecker interface {
	IdleUsable() bool
}

// IdleUsable reports false if an idle connection has unread input or is closed. True is best-effort,
// and codecs that cannot inspect their transport report true.
func (T *Conn) IdleUsable() bool {
	if c, ok := T.codec.(idleChecker); ok {
		return c.IdleUsable()
	}
	return true
}
