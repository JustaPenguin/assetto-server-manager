package when

import (
	"errors"
	"sync"
	"time"
)

var (
	// Resolution is how often each When is checked.
	Resolution = time.Second

	timers = make(map[*Timer]bool)
	mutex  = sync.Mutex{}
	once   = sync.Once{}
)

type Timer struct {
	fn func()
	t  time.Time
}

func newTimer(t time.Time, fn func()) *Timer {
	r := &Timer{
		t:  t,
		fn: fn,
	}

	return r
}

func (t *Timer) Stop() {
	mutex.Lock()
	delete(timers, t)
	mutex.Unlock()
}

var ErrTimeInPast = errors.New("when: time specified is in the past")

func When(t time.Time, fn func()) (*Timer, error) {
	if t.Before(time.Now()) {
		return nil, ErrTimeInPast
	}

	once.Do(func() {
		go func() {
			ticker := time.NewTicker(Resolution)

			for {
				select {
				case t := <-ticker.C:
					var toStop []*Timer

					mutex.Lock()
					for timer := range timers {
						if t.Round(Resolution).Equal(timer.t.Round(Resolution)) {
							go timer.fn()
							toStop = append(toStop, timer)
						}
					}
					mutex.Unlock()

					for _, timer := range toStop {
						timer.Stop()
					}
				}
			}
		}()
	})

	x := newTimer(t, fn)

	mutex.Lock()
	timers[x] = true
	mutex.Unlock()

	return x, nil
}
