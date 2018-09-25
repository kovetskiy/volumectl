package main

import "time"

type Deadline struct {
	duration   time.Duration
	onTimedOut func()
}

func (deadline *Deadline) do(job func()) {
	done := make(chan struct{})
	go func() {
		job()
		done <- struct{}{}
	}()

	after := time.After(deadline.duration)
	select {
	case <-after:
		deadline.onTimedOut()
		deadline.do(job)
	case <-done:
		return
	}
}
