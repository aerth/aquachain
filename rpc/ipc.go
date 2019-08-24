// Copyright 2018 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"fmt"
	"net"

	"gitlab.com/aquachain/aquachain/common/log"
)

// CreateIPCListener creates an listener, on Unix platforms this is a unix socket, on
// Windows this is a named pipe
func CreateIPCListener(endpoint string) (net.Listener, error) {
	return ipcListen(endpoint)
}

// ServeListener accepts connections on l, serving JSON-RPC on them.
func (srv *Server) ServeListener(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		log.Trace(fmt.Sprint("accepted conn", conn.RemoteAddr()))
		go srv.ServeCodec(NewJSONCodec(conn), OptionMethodInvocation|OptionSubscriptions)
	}
}
