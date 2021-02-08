package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gonutz/w32"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"github.com/winglq/eyenurse/monitor"
	"github.com/winglq/eyenurse/statemachine"
)

type Manager struct {
	stopRestCh         chan struct{}
	Done               chan struct{}
	wg                 sync.WaitGroup
	window             *walk.MainWindow
	nonMainMonitorWins []*walk.MainWindow
	te                 *walk.TextEdit
	sm                 *statemachine.EyeNurseSM
	o                  statemachine.Option
}

func NewManager() *Manager {
	m := &Manager{
		stopRestCh:         make(chan struct{}),
		Done:               make(chan struct{}),
		window:             new(walk.MainWindow),
		nonMainMonitorWins: []*walk.MainWindow{},
	}
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

func (m *Manager) setWinsVisible(visible bool) {
	m.window.SetVisible(visible)
	m.SetNonMainVisible(visible)
}

func (m *Manager) onStateChange(s statemachine.State) {
	switch s {
	case statemachine.Work:
		close(m.stopRestCh)
		m.stopRestCh = make(chan struct{})
		m.setWinsVisible(false)
	case statemachine.Rest:
		m.setWinsVisible(true)
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			t := time.NewTicker(time.Second)
			left := m.o.RestSeconds
			m.te.SetText(fmt.Sprintf("%d", left))
			for {
				select {
				case <-t.C:
					left--
					m.te.SetText(fmt.Sprintf("%d", left))
					if left == 0 {
						return
					}
				case <-m.Done:
					fmt.Println("handleRest Done")
					return
				case <-m.stopRestCh:
					fmt.Println("handleRest Stop")
					return
				}
			}
		}()
	case statemachine.Delay:
		close(m.stopRestCh)
		m.stopRestCh = make(chan struct{})
		m.setWinsVisible(false)
	case statemachine.Closing:
		os.Exit(0)
	}
}

func (m *Manager) AddNonMain(r w32.RECT) *walk.MainWindow {
	var w *walk.MainWindow
	MainWindow{
		Visible:  false,
		AssignTo: &w,
		Layout:   VBox{},
		Children: []Widget{},
	}.Create()
	win.SetWindowLong(m.window.Handle(), win.GWL_STYLE, win.WS_BORDER)
	win.SetWindowPos(
		w.Handle(),
		win.HWND_TOPMOST,
		r.Left,
		r.Top,
		r.Width(),
		r.Height(),
		win.SWP_FRAMECHANGED,
	)
	log.Printf("non main %d %d %d %d", r.Left, r.Top, r.Width(), r.Height())
	return w

}

func (m *Manager) CreateNonMain() {
	rects := monitor.GetMonitorRects()[1:]
	for _, r := range rects {
		w := m.AddNonMain(r)
		m.nonMainMonitorWins = append(m.nonMainMonitorWins, w)
	}
}

func (m *Manager) SetNonMainVisible(visible bool) {
	rects := monitor.GetMonitorRects()[1:]
	if len(rects) > len(m.nonMainMonitorWins) {
		starts := len(rects) - len(m.nonMainMonitorWins)
		for s := starts; s < len(rects); s++ {
			w := m.AddNonMain(rects[s])
			m.nonMainMonitorWins = append(m.nonMainMonitorWins, w)
		}
	}
	for i, r := range rects {
		win.SetWindowPos(
			m.nonMainMonitorWins[i].Handle(),
			win.HWND_TOPMOST,
			r.Left,
			r.Top,
			r.Width(),
			r.Height(),
			win.SWP_FRAMECHANGED,
		)
		m.nonMainMonitorWins[i].SetVisible(visible)
	}
}

func (m *Manager) CreateMain() {
	MainWindow{
		Visible:  false,
		AssignTo: &m.window,
		Layout:   VBox{},
		Children: []Widget{
			TextEdit{AssignTo: &m.te, ReadOnly: true},
			PushButton{
				Text: "Delay",
				OnClicked: func() {
					m.sm.SendEvent(statemachine.DelayRest)
				},
			},
			PushButton{
				Text: "Skip",
				OnClicked: func() {
					m.sm.SendEvent(statemachine.SkipRest)
				},
			},
			PushButton{
				Text: "Quit",
				OnClicked: func() {
					m.sm.SendEvent(statemachine.Quit)
				},
			},
		},
	}.Create()
	win.SetWindowLong(m.window.Handle(), win.GWL_STYLE, win.WS_BORDER)
	xScreen := win.GetSystemMetrics(win.SM_CXSCREEN)
	yScreen := win.GetSystemMetrics(win.SM_CYSCREEN)
	win.SetWindowPos(
		m.window.Handle(),
		win.HWND_TOPMOST,
		0,
		0,
		xScreen,
		yScreen,
		win.SWP_FRAMECHANGED,
	)
	log.Printf("main %d %d %d %d", 0, 0, xScreen, yScreen)
}

func (m *Manager) RunWindow() {
	m.window.Run()
}

func main() {
	m := NewManager()
	m.CreateMain()
	m.CreateNonMain()
	time.Sleep(time.Second)
	go m.sm.Run()
	m.RunWindow()
}
