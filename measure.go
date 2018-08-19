package main

import (
	"fmt"
	"sync"
	"time"
)

type measurement struct {
	title   []interface{}
	started time.Time
}

var (
	measurePool = &sync.Pool{
		New: func() interface{} {
			return &measurement{}
		},
	}
)

func measure(title ...interface{}) *measurement {
	measurement := measurePool.Get().(*measurement)
	measurement.title = title
	measurement.started = time.Now()
	return measurement
}

func (measurement *measurement) add(title ...interface{}) {
	measurement.title = append(measurement.title, title...)
}

func (measurement *measurement) stop() {
	finished := time.Now()

	fmt.Printf(
		"TIME %-30s %.2fms\n",
		fmt.Sprint(measurement.title),
		finished.Sub(measurement.started).Seconds()*1000.0,
	)

	measurePool.Put(measurement)
}

func withMeasure(title string, fn func()) {
	time := measure(title)
	fn()
	time.stop()
}
