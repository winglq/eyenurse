package statemachine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type State int

const (
	Work State = iota
	Rest
	Delay
	Closing
)

var stateMap = map[State]string{
	Work:    "work",
	Rest:    "rest",
	Delay:   "delay",
	Closing: "closing",
}

func (s State) String() string {
	return stateMap[s]
}

type Event int

const (
	WorkComplete = iota
	RestComplete
	DelayComplete
	DelayRest
	SkipRest
	Quit
)

var eventMap = map[Event]string{
	WorkComplete:  "work_complete",
	RestComplete:  "rest_complete",
	DelayComplete: "delay_complete",
	DelayRest:     "delay_rest",
	SkipRest:      "skip_rest",
	Quit:          "quit",
}

func (e Event) String() string {
	return eventMap[e]
}

type OnStateChange func(newState State)

type Option struct {
	WorkSeconds   time.Duration
	DelaySeconds  time.Duration
	RestSeconds   time.Duration
	StateChangeCB OnStateChange
}

type EyeNurseSM struct {
	state      State
	evtCH      chan Event
	done       chan struct{}
	wg         sync.WaitGroup
	o          Option
	cancelRest context.CancelFunc
}

func NewEyeNurseSM(o Option) *EyeNurseSM {
	sm := &EyeNurseSM{
		evtCH: make(chan Event),
		o:     o,
		done:  make(chan struct{}),
		state: Rest,
	}
	go func() {
		sm.SendEvent(RestComplete)
	}()
	return sm
}

func (sm *EyeNurseSM) Run() {
	sm.wg.Add(1)
	go sm.stateTranx()
	sm.wg.Wait()

}

func (sm *EyeNurseSM) SendEvent(e Event) {
	sm.evtCH <- e
}

func (sm *EyeNurseSM) stateTranx() {
	defer sm.wg.Done()
	for {
		select {
		case e := <-sm.evtCH:
			sm.processEvent(e)
		case <-sm.done:
			return
		}
	}
}

func (sm *EyeNurseSM) setState(s State) {
	sm.state = s
	switch s {
	case Work:
		sm.wg.Add(1)
		go func() {
			defer sm.wg.Done()
			select {
			case <-time.After(time.Second * sm.o.WorkSeconds):
				sm.evtCH <- WorkComplete
			case <-sm.done:
				return
			}
		}()
	case Rest:
		ctx := context.Background()
		ctx, sm.cancelRest = context.WithCancel(ctx)
		sm.wg.Add(1)
		go func() {
			defer sm.wg.Done()
			select {
			case <-time.After(time.Second * sm.o.RestSeconds):
				sm.evtCH <- RestComplete
			case <-sm.done:
				return
			case <-ctx.Done():
				return
			}
		}()
	case Delay:
		sm.wg.Add(1)
		go func() {
			defer sm.wg.Done()
			select {
			case <-time.After(time.Second * sm.o.DelaySeconds):
				sm.evtCH <- DelayComplete
			case <-sm.done:
				return
			}
		}()

	}
	sm.o.StateChangeCB(s)
}

func (sm *EyeNurseSM) processEvent(e Event) {
	switch sm.state {
	case Work:
		sm.processWorkState(e)
	case Rest:
		sm.processRestState(e)
	case Delay:
		sm.processDelayState(e)
	case Closing:
		sm.processClosingState(e)
	}
}

func (sm *EyeNurseSM) unhandledEvent(e Event) {
	panic(fmt.Sprintf("state %s, event %s", sm.state, e))
}

func (sm *EyeNurseSM) processWorkState(e Event) {
	switch e {
	case WorkComplete:
		sm.setState(Rest)
	default:
		sm.unhandledEvent(e)
	}
}

func (sm *EyeNurseSM) processRestState(e Event) {
	switch e {
	case DelayRest:
		sm.cancelRest()
		sm.cancelRest = nil
		sm.setState(Delay)
	case RestComplete:
		sm.cancelRest = nil
		sm.setState(Work)
	case SkipRest:
		sm.cancelRest()
		sm.cancelRest = nil
		sm.setState(Work)
	case Quit:
		close(sm.done)
		sm.setState(Closing)
	default:
		sm.unhandledEvent(e)
	}
}

func (sm *EyeNurseSM) processDelayState(e Event) {
	switch e {
	case DelayComplete:
		sm.setState(Rest)
	default:
		sm.unhandledEvent(e)
	}
}

func (sm *EyeNurseSM) processClosingState(e Event) {
	sm.unhandledEvent(e)
}
