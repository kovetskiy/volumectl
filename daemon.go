package main

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/reconquest/karma-go"
	"github.com/reconquest/sign-go"
)

func handleDaemon(args map[string]interface{}) error {
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

	defer pulse.Close()
	defer socket.Close()

	go sign.Notify(func(os.Signal) bool {
		err := socket.Close()
		if err != nil {
			logger.Error(err, "unable to gracefully stop listening unix socket")
		}

		pulse.Close()

		return false
	}, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	logger.Infof("listening for connections")

	serveDaemon(pulse, socket)

	return nil
}

func serveDaemon(pulse *Pulse, socket *Socket) {
	for {
		conn, err := socket.Accept()
		if err != nil {
			break
		}

		go serveDaemonConnection(pulse, socket, conn)
	}
}

func serveDaemonConnection(pulse *Pulse, socket *Socket, conn net.Conn) {
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
		var volume float32
		var retried bool

		for {
			volume, err = pulse.ChangeVolume(raw.(*PacketChange).Diff)
			if isNoSuchEntityError(err) {
				if retried {
					break
				}

				err := pulse.Reconnect()
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
			reply = PacketError{err.Error()}
		} else {
			reply = PacketVolume{volume}
		}

	case SignatureGet:
		reply = PacketVolume{pulse.GetVolume()}

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
