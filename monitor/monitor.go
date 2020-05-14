package monitor

import (
	"syscall"

	"github.com/gonutz/w32"
)

var Rects []w32.RECT

func init() {
	Rects = []w32.RECT{}
	r := GetMonitorRects()
	for _, e := range r {
		Rects = append(Rects, *e)
	}
}

// GetMonitorRects Get all monitor recats
func GetMonitorRects() []*w32.RECT {
	r := []*w32.RECT{}
	callback := func(arg1 w32.HMONITOR, arg2 w32.HDC, arg3 *w32.RECT, arg4 w32.LPARAM) uintptr {
		r = append(r, arg3)
		return 1
	}
	w32.EnumDisplayMonitors(w32.HDC(0), nil, syscall.NewCallback(callback), uintptr(0))
	return r
}
