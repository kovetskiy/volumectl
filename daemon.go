package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/reconquest/karma-go"
	"github.com/reconquest/sign-go"
)

type Daemon struct {
	pulse    *Pulse
	deadline *Deadline
}

func handleDaemon(args map[string]interface{}) error {
	deadlineMs, err := strconv.ParseInt(args["--deadline"].(string), 10, 64)
	if err != nil {
		return karma.Format(
			err,
			"unable to parse deadline",
		)
	}

	deadline := &Deadline{
		duration: time.Duration(deadlineMs) * time.Millisecond,
	}

	logger.Infof("initializing pulseaudio connection")

	pulse, err := initPulse()
	if err != nil {
		return karma.Format(
			err,
			"unable to initialize pulseaudio",
		)
	}

	defer pulse.Close()

	logger.Infof("initializing unix socket")

	socket, err := listenSocket(args["--socket"].(string))
	if err != nil {
		return karma.Format(
			err,
			"unable to initialize unix socket: %s", args["--socket"].(string),
		)
	}

	var tcp net.Listener
	if addr, ok := args["--tcp"].(string); ok {
		logger.Infof("initializing tcp socket")

		tcp, err = net.Listen("tcp", addr)
		if err != nil {
			return karma.Format(
				err,
				"unable to initialize tcp socket: %s", addr,
			)
		}
	}

	defer pulse.Close()
	defer socket.Close()

	if tcp != nil {
		defer tcp.Close()
	}

	go sign.Notify(func(os.Signal) bool {
		err := socket.Close()
		if err != nil {
			logger.Error(err, "unable to gracefully stop listening unix socket")
		}

		if tcp != nil {
			err = tcp.Close()
			logger.Error(err, "unable to gracefully stop listening tcp socket")
		}

		pulse.Close()

		return false
	}, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	logger.Infof("listening for connections")

	serveDaemon(pulse, deadline, socket, tcp)

	return nil
}

func serveDaemon(pulse *Pulse, deadline *Deadline, listeners ...net.Listener) {
	deadline.onTimedOut = func() {
		logger.Errorf(
			"operation timed out after %v, resetting connection and retrying",
			deadline.duration,
		)

		err := pulse.Reconnect()
		if err != nil {
			logger.Error(err)
		}

		logger.Warningf("re-established connection to pulseaudio")
	}

	daemon := &Daemon{
		pulse:    pulse,
		deadline: deadline,
	}

	wg := sync.WaitGroup{}
	for _, listener := range listeners {
		if listener == nil {
			continue
		}

		wg.Add(1)
		go func(listener net.Listener) {
			defer wg.Done()
			for {
				conn, err := listener.Accept()
				if err != nil {
					break
				}

				go daemon.serve(conn)
			}
		}(listener)
	}

	wg.Wait()
}

func (daemon *Daemon) serve(conn net.Conn) {
	daemon.pulse.Lock()
	defer daemon.pulse.Unlock()

	serving := measure("serve connection")

	defer serving.stop()
	defer conn.Close()

	var raw Packetable
	var err error
	withMeasure("unpacking", func() {
		raw, err = unpack(conn)
	})
	if err != nil {
		logger.Error(karma.Format(err, "unable to decode packet"))
		return
	}

	var reply Packetable
	switch raw.Signature() {
	case SignatureChange:
		daemon.deadline.do(func() {
			reply = daemon.changeVolume(raw.(*PacketChange))
		})

	case SignatureGet:
		reply = PacketVolume{daemon.pulse.GetVolume()}

	default:
		reply = PacketError{
			fmt.Sprintf("unknown packet signature: %s", raw.Signature()),
		}
	}

	if reply != nil {
		withMeasure("packing", func() {
			err = pack(conn, reply)
			if err != nil {
				err = karma.Format(err, "unable to encode packet")
			}
		})
	}

	if err != nil {
		logger.Error(err)
	}
}

func (daemon *Daemon) changeVolume(packet *PacketChange) Packetable {
	var volume float32
	var retried bool
	var err error

	for {
		volume, err = daemon.pulse.ChangeVolume(packet.Diff)
		if isNoSuchEntityError(err) {
			if retried {
				break
			}

			err := daemon.pulse.Reconnect()
			if err != nil {
				err = karma.Format(
					err,
					"unable to reconnect to pulseaudio",
				)
				break
			}

			retried = true
		}

		break
	}

	if err != nil {
		return PacketError{err.Error()}
	}

	return PacketVolume{volume}
}
