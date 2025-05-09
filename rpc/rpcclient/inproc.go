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
	"context"
	"net"

	"gitlab.com/aquachain/aquachain/rpc"
)

// NewInProcClient attaches an in-process connection to the given RPC server.
func DialInProc(ctx context.Context, handler *rpc.Server) *Client {
	if handler == nil {
		panic("InProc handler is nil")
	}
	c, err := newClient(ctx, func(context.Context) (net.Conn, error) {
		p1, p2 := net.Pipe()
		go handler.ServeCodec(p2.RemoteAddr().String(), rpc.NewJSONCodec(p1), rpc.OptionMethodInvocation|rpc.OptionSubscriptions)
		return p2, nil
	})
	if err != nil {
		panic(err.Error())
	}
	return c
}
