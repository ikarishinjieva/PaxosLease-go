package paxosLease

import (
	"time"
)

type tick struct {
	callback func()
	quitChan chan bool
}

func newTick(callback func()) *tick {
	ret := tick{}
	ret.callback = callback
	ret.quitChan = make(chan bool, 1)
	return &ret
}

func (t *tick) start(ms int) *tick {
	go func() {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		select {
		case <-t.quitChan:
			return
		default:
			t.callback()
		}
	}()
	return t
}

func (t *tick) stop() {
	t.quitChan <- true
}
