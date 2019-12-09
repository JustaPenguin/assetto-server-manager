package when

import (
	"errors"
	"sync"
	"time"
)

var (
	// Resolution is how often each When is checked.
	Resolution = time.Second

	events = make(map[chan struct{}]runnable)
	mutex  = sync.Mutex{}
	once   = sync.Once{}
)

type runnable struct {
	fn func()
	t  time.Time
}

var ErrTimeInPast = errors.New("when: time specified is in the past")

func When(t time.Time, fn func()) (chan<- struct{}, error) {
	if t.Before(time.Now()) {
		return nil, ErrTimeInPast
	}

	once.Do(func() {
		go func() {
			ticker := time.NewTicker(Resolution)

			for {
				select {
				case t := <-ticker.C:
					mutex.Lock()
					for _, event := range events {
						if t.Round(Resolution).Equal(event.t.Round(Resolution)) {
							go event.fn()
						}
					}
					mutex.Unlock()
				}
			}
		}()
	})

	ch := make(chan struct{})

	mutex.Lock()
	events[ch] = runnable{t: t, fn: fn}
	mutex.Unlock()

	go func() {
		select {
		case <-ch:
			mutex.Lock()
			delete(events, ch)
			mutex.Unlock()
			return
		}
	}()

	return ch, nil
}
