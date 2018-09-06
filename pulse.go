package main

import (
	"sync"

	"github.com/godbus/dbus"
	native "github.com/kovetskiy/pulseaudio"
	bus "github.com/kovetskiy/pulseaudio-bus"
	"github.com/reconquest/karma-go"
)

var (
	_ PulseSubscriber = (*Pulse)(nil)
)

type PulseSubscriber interface {
	bus.OnNewSink
	bus.OnSinkRemoved
	bus.OnDeviceActivePortUpdated
}

type Pulse struct {
	client *native.Client
	info   *native.Server
	bus    *bus.Client
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

	pulse.Once = sync.Once{}

	return pulse.initNative()
}

func initPulse() (*Pulse, error) {
	pulse := &Pulse{}

	err := pulse.initNative()
	if err != nil {
		return nil, err
	}

	err = pulse.initBus()
	if err != nil {
		err = bus.LoadModule()
		if err != nil {
			logger.Error(karma.Format(err, "unable to load pulseaudio dbus module"))
		}

		err = pulse.initBus()
	}
	if err != nil {
		return nil, err
	}

	go pulse.bus.Listen()

	return pulse, nil
}

func (pulse *Pulse) initBus() error {
	var err error
	withMeasure("pulse:dbus:connect", func() {
		pulse.bus, err = bus.New()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to connect to dbus",
		)
	}

	var regErr error
	withMeasure("pulse:dbus:register-handlers", func() {
		errs := pulse.bus.Register(pulse)
		if errs != nil {
			for _, err := range errs {
				logger.Error(err)
			}

			regErr = karma.Format(
				err,
				"unable to register dbus client",
			)
		}
	})

	return regErr
}

func (pulse *Pulse) initNative() error {
	var err error
	withMeasure("pulse:connect", func() {
		pulse.client, err = native.NewClient()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to connect to pulseaudio socket",
		)
	}

	withMeasure("pulse:get-server-info", func() {
		pulse.info, err = pulse.client.ServerInfo()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to get server info",
		)
	}

	withMeasure("pulse:get-volume", func() {
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
	withMeasure("pulse:set-sink-volume", func() {
		err = pulse.client.SetSinkVolume(pulse.info.DefaultSink, volume)
	})

	if err == nil {
		pulse.volume = volume
	}

	pulse.Unlock()

	return volume, err
}

// NewSink is called when a sink is added.
func (pulse *Pulse) NewSink(path dbus.ObjectPath) {
	logger.Infof("dbus: new sink added, reconnecting")

	err := pulse.Reconnect()
	if err != nil {
		logger.Error(karma.Format(err, "unable to reconnect to pulseaudio"))
	}
}

// SinkRemoved is called when a sink is removed.
func (pulse *Pulse) SinkRemoved(path dbus.ObjectPath) {
	logger.Infof("dbus: sink removed, reconnecting")

	err := pulse.Reconnect()
	if err != nil {
		logger.Error(karma.Format(err, "unable to reconnect to pulseaudio"))
	}
}

func (pulse *Pulse) DeviceActivePortUpdated(dbus.ObjectPath, dbus.ObjectPath) {
	logger.Infof("dbus: sink active port updated, reconnecting")

	err := pulse.Reconnect()
	if err != nil {
		logger.Error(karma.Format(err, "unable to reconnect to pulseaudio"))
	}
}

func isNoSuchEntityError(err error) bool {
	if err != nil {
		if specific, ok := err.(*native.Error); ok {
			if specific.Code == 0x5 {
				return true
			}
		}
	}

	return false
}
