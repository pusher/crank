package netutil

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
)

// Utility to open a file on a port, path or file descriptor
func BindFile(addr string) (file *os.File, err error) {
	u, err := url.Parse(addr)
	if err != nil {
		return
	}

	switch u.Scheme {
	case "fd":
		var fd uint64
		fd, err = strconv.ParseUint(u.Host, 10, 8)
		if err != nil {
			return
		}
		// NOTE: The name argument doesn't really matter apparently
		file = os.NewFile(uintptr(fd), addr)
	case "unix", "unixpacket":
		var laddr *net.UnixAddr
		var listener *net.UnixListener
		path := u.Host + u.Path

		laddr, err = net.ResolveUnixAddr(u.Scheme, path)
		if err != nil {
			return
		}

		// In case crank crashes the socket file wouldn't be cleaned up.
		// We prefer having two crank running on the same socket file than
		// none because the file exists.
		os.Remove(path)

		listener, err = net.ListenUnix(laddr.Network(), laddr)
		if err != nil {
			return
		}

		file, err = listener.File()
	case "tcp", "tcp4", "tcp6":
		var laddr *net.TCPAddr
		var listener *net.TCPListener

		laddr, err = net.ResolveTCPAddr(u.Scheme, u.Host)
		if err != nil {
			return
		}

		listener, err = net.ListenTCP(laddr.Network(), laddr)
		if err != nil {
			return
		}

		// Closing the listener doesn't affect the file and reversely.
		// http://golang.org/pkg/net/#TCPListener.File
		file, err = listener.File()
	default:
		err = fmt.Errorf("Unsupported scheme: %s", u.Scheme)
	}

	return
}
