package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/reconquest/karma-go"
)

func handleClient(args map[string]interface{}) error {
	var percent float32
	if raw, ok := args["<percent>"].(string); ok {
		value, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return karma.Format(
				err,
				"unable to parse percent value: %s", raw,
			)
		}

		percent = float32(value) / 100
	}

	conn, err := net.Dial("unix", args["--socket"].(string))
	if err != nil {
		return karma.Format(
			err,
			"unable to connect to unix socket: %s",
			args["--socket"].(string),
		)
	}

	defer conn.Close()

	switch {
	case args["up"].(bool):
		err = pack(conn, PacketChange{percent})
	case args["down"].(bool):
		err = pack(conn, PacketChange{-percent})
	case args["get"].(bool):
		err = pack(conn, PacketGet{})
	default:
		return fmt.Errorf("unexpected args")
	}
	if err != nil {
		return err
	}

	raw, err := unpack(conn)
	if err != nil {
		return karma.Format(
			err,
			"unable to decode packet",
		)
	}

	switch raw.Signature() {
	case SignatureVolume:
		fmt.Println(raw.(*PacketVolume).Value * 100)
	case SignatureError:
		return karma.Format(
			raw.(*PacketError).Error,
			"the daemon sent an error",
		)
	default:
		return fmt.Errorf("unexpected packet signature: %s", raw.Signature())
	}

	return nil
}
