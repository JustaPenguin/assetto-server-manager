package when

import (
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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

			for tick := range ticker.C {
				var toStop []*Timer

				mutex.Lock()
				for timer := range timers {
					if tick.Round(Resolution).Equal(timer.t.Round(Resolution)) || tick.Round(Resolution).After(timer.t.Round(Resolution)) {
						logrus.Debugf("Starting scheduled event (is now %s)", timer.t)
						go timer.fn()
						toStop = append(toStop, timer)
					}
				}
				mutex.Unlock()

				for _, timer := range toStop {
					timer.Stop()
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
