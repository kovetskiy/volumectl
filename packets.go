package main

import (
	"net"

	"github.com/vmihailenco/msgpack"
)

func registerPackets() {
	msgpack.RegisterExt(23, PacketChange{})
	msgpack.RegisterExt(24, PacketVolume{})
	msgpack.RegisterExt(25, PacketError{})
	msgpack.RegisterExt(26, PacketGet{})
}

func pack(conn net.Conn, packet Packetable) error {
	return msgpack.NewEncoder(conn).Encode(packet)
}

func unpack(conn net.Conn) (Packetable, error) {
	var packet Packetable
	err := msgpack.NewDecoder(conn).Decode(&packet)
	return packet, err
}

type Packetable interface {
	Signature() string
}

const SignatureChange = "change"

type PacketChange struct {
	Diff float32
}

func (packet PacketChange) Signature() string {
	return SignatureChange
}

const SignatureVolume = "volume"

type PacketVolume struct {
	Value float32
}

func (packet PacketVolume) Signature() string {
	return SignatureVolume
}

const SignatureGet = "get"

type PacketGet struct {
}

func (packet PacketGet) Signature() string {
	return SignatureGet
}

const SignatureError = "error"

type PacketError struct {
	Error string
}

func (packet PacketError) Signature() string {
	return SignatureError
}
