package paxosLease

import (
	"time"
)

type tick struct {
	callback   func()
	quitChan   chan bool
	expireTime time.Time
}

func newTick(callback func()) *tick {
	ret := tick{}
	ret.callback = callback
	ret.quitChan = make(chan bool, 1)
	return &ret
}

func (t *tick) start(ms int) *tick {
	t.expireTime = time.Now().Add(time.Duration(ms) * time.Millisecond)
	go func() {
		select {
		case <-t.quitChan:
			return
		case <-time.After(time.Duration(ms) * time.Millisecond):
			t.callback()
		}
	}()
	return t
}

func (t *tick) stop() {
	t.quitChan <- true
}
