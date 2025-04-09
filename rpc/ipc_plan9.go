package rpc

import (
	"fmt"
	"net"
)

// ipcListen will create a named pipe on the given endpoint.
func ipcListen(endpoint string) (net.Listener, error) {
	return nil, fmt.Errorf("not implemented on plan9")
}
