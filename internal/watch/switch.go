package watch

import "time"

type retry struct {
	lastStatus    bool
	currentStatus bool
	count         int
	lastTime      time.Time
}

func newRetry() *retry {
	return &retry{
		count: 0,
		//lastTime: time.Now(),
	}
}

func (r *retry) check(check bool, ok func() error, bad func() error) error {
	if check {
		if r.lastStatus {
			r.count += 1
		} else {
			r.count = 0
		}
		r.lastStatus = true
		if !r.currentStatus && r.count > 3 {
			if time.Since(r.lastTime) < time.Minute*60 {
				return nil
			}
			if err := ok(); err == nil {
				r.currentStatus = true
				r.lastTime = time.Now()
			}
		}
	} else {
		if !r.lastStatus {
			r.count += 1
		} else {
			r.count = 0
		}
		r.lastStatus = false
		if r.currentStatus && r.count > 3 {
			if err := bad(); err == nil {
				r.currentStatus = false
				r.lastTime = time.Now()
			}
		}
	}
	return nil
}
