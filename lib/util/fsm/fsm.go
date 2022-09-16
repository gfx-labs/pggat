package fsm

import (
	"fmt"
	"sync"
)

// TransitionRuleSet is a set of allowed transitions. This uses map of struct{}
// to implement a set.
type TransitionRuleSet map[string]struct{}

// Copy copies the TransitionRuleSet in to a different TransitionRuleSet.
func (trs TransitionRuleSet) Copy() TransitionRuleSet {
	srt := make(TransitionRuleSet)

	for rule, value := range trs {
		srt[rule] = value
	}

	return srt
}

// CallbackHandler is an interface type defining the interface for receiving callbacks.
type CallbackHandler interface {
	StateTransitionCallback(string) error
}

// Machine is the state machine.
type Machine struct {
	state string
	mu    sync.RWMutex

	transitions map[string]TransitionRuleSet

	callback     CallbackHandler
	syncCallback bool
}

func (m *Machine) Clone() *Machine {
	return &Machine{
		state:        m.state,
		transitions:  m.transitions,
		callback:     m.callback,
		syncCallback: m.syncCallback,
	}
}

// CurrentState returns the machine's current state. If the State returned is
// "", then the machine has not been given an initial state.
func (m *Machine) CurrentState() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.state
}

// StateTransitionRules returns the allowed states for
func (m *Machine) StateTransitionRules(state string) (TransitionRuleSet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.transitions == nil {
		return nil, newErrorStruct("the machine has not been fully initialized", ErrorMachineNotInitialized)
	}

	// ensure the state has been registered
	if _, ok := m.transitions[state]; !ok {
		return nil, newErrorStruct(fmt.Sprintf("state %s has not been registered", state), ErrorStateUndefined)
	}

	return m.transitions[state].Copy(), nil
}

// AddStateTransitionRules is a function for adding valid state transitions to the machine.
// This allows you to define which states any given state can be transitioned to.
func (m *Machine) AddStateTransitionRules(sourceState string, destinationStates ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// if the transitions map is nil, we need to allocate it
	if m.transitions == nil {
		m.transitions = make(map[string]TransitionRuleSet)
	}

	// if the map for the source state does not exist, allocate it
	if m.transitions[sourceState] == nil {
		m.transitions[sourceState] = make(TransitionRuleSet)
	}

	// get a reference to the map we care about
	// avoids doing the map lookup for each iteration
	mp := m.transitions[sourceState]

	for _, dest := range destinationStates {
		mp[dest] = struct{}{}
	}

	return nil
}

// SetStateTransitionCallback for the state transition. This is meant to send
// callbacks back to the consumer for state changes. The callback only sends the
// new state. The synchonous parameter indicates whether the callback is done
// synchronously with the StateTransition() call.
func (m *Machine) SetStateTransitionCallback(callback CallbackHandler, synchronous bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callback = callback
	m.syncCallback = synchronous

	return nil
}

// StateTransition triggers a transition to the toState. This function is also
// used to set the initial state of machine.
//
// Before you can transition to any state, even for the initial, you must define
// it with AddStateTransition(). If you are setting the initial state, and that
// state is not define, this will return an ErrInvalidInitialState error.
//
// When transitioning from a state, this function will return an error either
// if the state transition is not allowed, or if the destination state has
// not been defined. In both cases, it's seen as a non-permitted state transition.
func (m *Machine) StateTransition(toState string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// if this is nil we cannot assume any state
	if m.transitions == nil {
		return newErrorStruct("the machine has no states added", ErrorMachineNotInitialized)
	}

	// if the state is nothing, this is probably the initial state
	if m.state == "" {
		// if the state is not defined, it's invalid
		if _, ok := m.transitions[toState]; !ok {
			return newErrorStruct("the initial state has not been defined within the machine", ErrorStateUndefined)
		}

		// set the state
		m.state = toState
		return nil
	}

	// if we are not permitted to transition to this state...
	if _, ok := m.transitions[m.state][toState]; !ok {
		return newErrorStruct(fmt.Sprintf("transition from state %s to %s is not permitted", m.state, toState), ErrorTransitionNotPermitted)
	}

	// if the destination state was not defined...
	if _, ok := m.transitions[toState]; !ok {
		return newErrorStruct(fmt.Sprintf("state %s has not been registered", toState), ErrorStateUndefined)
	}

	m.state = toState

	if m.callback != nil {
		if m.syncCallback {
			// do not return the error
			// this may be reconsidered
			m.callback.StateTransitionCallback(toState)
		} else {
			// spin off the callback
			go func() { m.callback.StateTransitionCallback(toState) }()
		}
	}

	return nil
}

type ErrorCode uint

func (e ErrorCode) String() string {
	switch e {
	case ErrorMachineNotInitialized:
		return "MachineNotInitialized"
	case ErrorTransitionNotPermitted:
		return "TransitionNotPermitted"
	case ErrorStateUndefined:
		return "StateUndefined"
	default:
		return "Unknown"
	}
}

const (
	// ErrorUnknown is the default value
	ErrorUnknown ErrorCode = iota

	// ErrorMachineNotInitialized is an error returned when actions are taken on
	// a machine before it has been initialized. A machine is initialized by
	// adding at least one state and setting it as the initial state.
	ErrorMachineNotInitialized

	// ErrorTransitionNotPermitted is the error returned when trying to
	// transition to an invalid state. In other words, the machine is not
	// permitted to transition from the current state to the one requested.
	ErrorTransitionNotPermitted

	// ErrorStateUndefined is the error returned when the requested state is
	// not defined within the machine.
	ErrorStateUndefined
)

// Error is the struct representing internal errors.
// This implements the error interface
type Error struct {
	message string
	code    ErrorCode
}

// newErrorStruct uses messge and code to create an *Error struct. The *Error
// struct implements the 'error' interface, so it should be able to be used
// wherever 'error' is expected.
func newErrorStruct(message string, code ErrorCode) *Error {
	return &Error{
		message: message,
		code:    code,
	}
}

// Message returns the error message.
func (e *Error) Message() string { return e.message }

// Code returns the error code.
func (e *Error) Code() ErrorCode { return e.code }

func (e *Error) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.code, e.code, e.message)
}
