package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
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
	te                         *walk.TextEdit
}

func NewManager() *Manager {
	return &Manager{
		c:          NewDefaultConfig(),
		wChan:      make(chan struct{}),
		rChan:      make(chan struct{}),
		dChan:      make(chan struct{}),
		sChan:      make(chan struct{}),
		stopRestCh: make(chan struct{}),
		Done:       make(chan struct{}),
		window:     new(walk.MainWindow),
	}
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
	m.window.SetVisible(false)
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
	m.window.SetVisible(false)
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
	m.window.SetVisible(true)
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
	m.Run()
	m.RunWindow()
}
