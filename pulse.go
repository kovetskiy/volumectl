package main

import (
	"sync"

	"github.com/kovetskiy/pulseaudio"
	"github.com/reconquest/karma-go"
)

type Pulse struct {
	client *pulseaudio.Client
	info   *pulseaudio.Server
	volume float32
	sync.Mutex
	sync.Once
}

func (pulse *Pulse) Close() {
	pulse.Once.Do(func() {
		withMeasure(
			"pulse:disconnect",
			pulse.client.Close,
		)
	})
}

func (pulse *Pulse) Reconnect() error {
	pulse.Lock()
	defer pulse.Unlock()

	pulse.Close()

	return pulse.init()
}

func initPulse() (*Pulse, error) {
	pulse := &Pulse{}

	err := pulse.init()
	if err != nil {
		return nil, err
	}

	return pulse, nil
}

func (pulse *Pulse) init() error {
	var err error
	withMeasure("pulse:connect", func() {
		pulse.client, err = pulseaudio.NewClient()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to connect to pulseaudio socket",
		)
	}

	withMeasure("get server info", func() {
		pulse.info, err = pulse.client.ServerInfo()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to get server info",
		)
	}

	withMeasure("get volume", func() {
		pulse.volume, err = pulse.client.Volume()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to get current volume level",
		)
	}

	return nil
}

func (pulse *Pulse) GetVolume() float32 {
	pulse.Lock()
	value := pulse.volume
	pulse.Unlock()
	return value
}

func (pulse *Pulse) ChangeVolume(diff float32) (float32, error) {
	pulse.Lock()

	volume := pulse.volume + diff

	var err error
	withMeasure("pulse: set-sink-volume", func() {
		err = pulse.client.SetSinkVolume(pulse.info.DefaultSink, volume)
	})

	if err == nil {
		pulse.volume = volume
	}

	pulse.Unlock()

	return volume, err
}

func isNoSuchEntityError(err error) bool {
	if err != nil {
		if specific, ok := err.(*pulseaudio.Error); ok {
			if specific.Code == 0x5 {
				return true
			}
		}
	}

	return false
}
