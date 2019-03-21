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

	bus.OnNewPlaybackStream
	bus.OnFallbackSinkUpdated
	bus.OnFallbackSinkUnset
	//bus.OnNewSink
	//bus.OnSinkRemoved
	//bus.OnNewPlaybackStream
	bus.OnPlaybackStreamRemoved
	//bus.OnDeviceVolumeUpdated
	bus.OnDeviceMuteUpdated
	//bus.OnStreamVolumeUpdated
	bus.OnStreamMuteUpdated
	//bus.OnDeviceActivePortUpdated
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
	pulse.Close()

	err := pulse.initNative()
	if err != nil {
		return err
	}

	pulse.Once = sync.Once{}

	return nil
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
	var client *native.Client
	withMeasure("pulse:connect", func() {
		client, err = native.NewClient()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to connect to pulseaudio socket",
		)
	}

	var info *native.Server
	withMeasure("pulse:get-server-info", func() {
		info, err = client.ServerInfo()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to get server info",
		)
	}

	var volume float32
	withMeasure("pulse:get-volume", func() {
		volume, err = client.Volume()
	})
	if err != nil {
		return karma.Format(
			err,
			"unable to get current volume level",
		)
	}

	pulse.client = client
	pulse.info = info
	pulse.volume = volume
	pulse.Once = sync.Once{}

	return nil
}

func (pulse *Pulse) GetVolume() float32 {
	value := pulse.volume
	return value
}

func (pulse *Pulse) ChangeVolume(diff float32) (float32, error) {
	volume := pulse.volume + diff

	var err error
	withMeasure("pulse:set-sink-volume", func() {
		err = pulse.client.SetSinkVolume(pulse.info.DefaultSink, volume)
	})

	if err == nil {
		pulse.volume = volume
	}

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

func (pulse *Pulse) FallbackSinkUpdated(dbus.ObjectPath) {
	logger.Infof("dbus: fallback sink updated, reconnecting")

	err := pulse.Reconnect()
	if err != nil {
		logger.Error(karma.Format(err, "unable to reconnect to pulseaudio"))
	}
}

func (pulse *Pulse) FallbackSinkUnset() {
	logger.Debugf("dbus: FallbackSinkUnset")
}

func (pulse *Pulse) NewPlaybackStream(dbus.ObjectPath) {
	logger.Debugf("dbus: NewPlaybackStream")
}

func (pulse *Pulse) PlaybackStreamRemoved(dbus.ObjectPath) {
	logger.Debugf("dbus: PlaybackStreamRemoved")
}

func (pulse *Pulse) DeviceMuteUpdated(dbus.ObjectPath, bool) {
	logger.Debugf("dbus: DeviceMuteUpdated")
}

func (pulse *Pulse) StreamMuteUpdated(dbus.ObjectPath, bool) {
	logger.Debugf("dbus: StreamMuteUpdated")
}
