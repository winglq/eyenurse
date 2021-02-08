package statemachine

import (
	"fmt"
	"testing"
	"time"
)

func TestSM(t *testing.T) {
	sm := NewEyeNurseSM(Option{
		WorkSeconds:  1,
		DelaySeconds: 1,
		RestSeconds:  2,
		StateChangeCB: func(s State) {
			fmt.Printf("state %s\n", s)
		},
	})
	go func() {
		<-time.After(2 * time.Second)
		sm.SendEvent(DelayRest)
	}()

	go func() {
		<-time.After(4 * time.Second)
		sm.SendEvent(Quit)
	}()
	sm.Run()
}
