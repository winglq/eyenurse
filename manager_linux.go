package main

import (
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/kbinani/screenshot"
	"github.com/winglq/eyenurse/statemachine"
)

type Manager struct {
	sm *statemachine.EyeNurseSM
	o  statemachine.Option
	w  fyne.Window
	a  fyne.App
}

func NewManager() *Manager {
	m := &Manager{
		a: app.New(),
	}
	m.w = m.a.NewWindow("eyenurse")
	bounds := screenshot.GetDisplayBounds(0)
	m.w.Resize(fyne.Size{float32(bounds.Dx()), float32(bounds.Dy())})
	m.w.SetContent(container.NewVBox(
		widget.NewButton("Delay", func() {
			m.sm.SendEvent(statemachine.DelayRest)
		}),
		widget.NewButton("Skip", func() {
			m.sm.SendEvent(statemachine.SkipRest)
		}),
		widget.NewButton("Exit", func() {
			m.sm.SendEvent(statemachine.Quit)
		}),
	))
	m.o = statemachine.Option{
		WorkSeconds:   60 * 45,
		DelaySeconds:  60 * 5,
		RestSeconds:   60 * 5,
		StateChangeCB: m.onStateChange,
	}
	//m.o = statemachine.Option{
	//	WorkSeconds:   5,
	//	DelaySeconds:  5,
	//	RestSeconds:   10,
	//	StateChangeCB: m.onStateChange,
	//}

	m.sm = statemachine.NewEyeNurseSM(m.o)
	return m
}

func (m *Manager) onStateChange(s statemachine.State) {
	switch s {
	case statemachine.Work:
		m.w.Hide()
	case statemachine.Rest:
		m.w.Show()
	case statemachine.Delay:
		m.w.Hide()
	case statemachine.Closing:
		os.Exit(0)

	}
}

func (m *Manager) Run() {
	go func() {
		time.Sleep(200 * time.Millisecond)
		m.w.Hide()
		m.sm.Run()
	}()
	m.w.ShowAndRun()
}
