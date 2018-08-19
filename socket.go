package main

import (
	"net"
	"os"
)

type Socket struct {
	path string
	net.Listener
}

func listenSocket(path string) (*Socket, error) {
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	return &Socket{
		path:     path,
		Listener: listener,
	}, nil
}

func (socket *Socket) Close() error {
	socket.Listener.Close()
	return os.Remove(socket.path)
}
