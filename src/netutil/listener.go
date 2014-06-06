package netutil

import (
	"fmt"
	"net"
	"os"
)

type unlinkListener struct {
	*net.UnixListener
}

func (l unlinkListener) Close() error {
	err := os.Remove(l.Addr().String())
	fmt.Println("ERR", err)
	return l.UnixListener.Close()
}

func UnlinkListener(l net.Listener) net.Listener {
	switch l2 := l.(type) {
	case *net.UnixListener:
		return unlinkListener{l2}
	default:
		return l
	}
}
