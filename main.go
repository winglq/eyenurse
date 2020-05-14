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
)

type Config struct {
	WorkDuration  time.Duration
	RestDuration  time.Duration
	DelayDuration time.Duration
}

func NewDefaultConfig() *Config {
	return &Config{
		WorkDuration:  45 * time.Minute,
		RestDuration:  5 * time.Minute,
		DelayDuration: 5 * time.Minute,
	}
}

func NewTestConfig() *Config {
	return &Config{
		WorkDuration:  1 * time.Second,
		RestDuration:  5 * time.Second,
		DelayDuration: 5 * time.Second,
	}
}

type Manager struct {
	c                          *Config
	wChan, rChan, dChan, sChan chan struct{}
	stopRestCh                 chan struct{}
	Done                       chan struct{}
	wg                         sync.WaitGroup
	window                     *walk.MainWindow
	nonMainMonitorWins         []*walk.MainWindow
	te                         *walk.TextEdit
}

func NewManager() *Manager {
	return &Manager{
		//c: NewTestConfig(),
		c:                  NewDefaultConfig(),
		wChan:              make(chan struct{}),
		rChan:              make(chan struct{}),
		dChan:              make(chan struct{}),
		sChan:              make(chan struct{}),
		stopRestCh:         make(chan struct{}),
		Done:               make(chan struct{}),
		window:             new(walk.MainWindow),
		nonMainMonitorWins: []*walk.MainWindow{},
	}
}

func (m *Manager) setWinsVisible(visible bool) {
	m.window.SetVisible(visible)
	m.SetNonMainVisible(visible)
}

func (m *Manager) handle() {
	m.wg.Done()
	for {
		select {
		case <-m.wChan:
			m.wg.Add(1)
			go m.handleWork()
		case <-m.rChan:
			m.wg.Add(1)
			go m.handleRest()
		case <-m.dChan:
			m.wg.Add(1)
			go m.handleDelay()
		case <-m.sChan:
			m.wg.Add(1)
			go m.handleSkip()
		case <-m.Done:
			fmt.Println("handle Done")
			return
		}
	}
}

func (m *Manager) handleSkip() {
	defer m.wg.Done()
	m.stopRestCh <- struct{}{}
	m.wChan <- struct{}{}
}

func (m *Manager) handleDelay() {
	defer m.wg.Done()
	m.setWinsVisible(false)
	m.stopRestCh <- struct{}{}
	select {
	case <-time.After(m.c.DelayDuration):
		m.rChan <- struct{}{}
	case <-m.Done:
		fmt.Println("handleWork Done")
		return
	}
}

func (m *Manager) handleWork() {
	defer m.wg.Done()
	m.setWinsVisible(false)
	select {
	case <-time.After(m.c.WorkDuration):
		m.rChan <- struct{}{}
	case <-m.Done:
		fmt.Println("handleWork Done")
		return
	}
}

func (m *Manager) handleRest() {
	defer m.wg.Done()
	m.setWinsVisible(true)
	t := time.NewTicker(time.Second)
	left := int(m.c.RestDuration.Seconds())
	m.te.SetText(fmt.Sprintf("%d", left))
	for {
		select {
		case <-t.C:
			left--
			m.te.SetText(fmt.Sprintf("%d", left))
			if left == 0 {
				m.wChan <- struct{}{}
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
}

func (m *Manager) Close() {
	close(m.Done)
	m.wg.Wait()
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

func (m *Manager) DestroyNonMain() {
	for _, w := range m.nonMainMonitorWins {
		w.Close()
	}
	m.nonMainMonitorWins = nil
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
					m.dChan <- struct{}{}
				},
			},
			PushButton{
				Text: "Skip",
				OnClicked: func() {
					m.sChan <- struct{}{}
				},
			},
			PushButton{
				Text: "Quit",
				OnClicked: func() {
					m.Close()
					os.Exit(0)
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

func (m *Manager) Run() {
	m.wg.Add(1)
	go m.handle()
	m.wChan <- struct{}{}
}

func main() {
	m := NewManager()
	m.CreateMain()
	m.CreateNonMain()
	m.Run()
	m.RunWindow()
}
