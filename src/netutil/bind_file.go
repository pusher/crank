package netutil

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// Utility to open a file on a port, path or file descriptor. Useful to bind
// but not use a specific socket (so it can be passed onto a child).
//
// Similar to net.Listen() except that it accepts a URI
func BindURI(uri string) (file *os.File, err error) {
	network, addr, err := uriToAddr(uri)
	if err != nil {
		return nil, err
	}

	switch network {
	case "fd":
		var fd uint64
		if fd, err = strconv.ParseUint(addr, 10, 8); err != nil {
			return
		}
		// The file name is arbitrary, here we use the uri
		file = os.NewFile(uintptr(fd), uri)
		return
	case "unix", "unixpacket", "unixgram":
		// In case a previous process didn't cleanup the socket properly.
		// We prefer of running the risk of having two processes than not being
		// able to bind. But only if the file is a socket.
		if fi, err2 := os.Lstat(addr); err2 == nil {
			if fi.Mode()&os.ModeSocket > 0 {
				os.Remove(addr)
			}
		}
	}

	switch network {
	case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
		var listener net.Listener
		if listener, err = net.Listen(network, addr); err != nil {
			return
		}
		// Closing the listener doesn't affect the file and reversely.
		// http://golang.org/pkg/net/#TCPListener.File
		file, err = listener.(filer).File()
	case "udp", "udp4", "udp6", "unixgram":
		var packetconn net.PacketConn
		if packetconn, err = net.ListenPacket(network, addr); err != nil {
			return
		}
		file, err = packetconn.(filer).File()
	default:
		err = fmt.Errorf("Unsupported network: %s", network)
	}
	return
}

// Like net.Dial but accepts a URI
func DialURI(uri string) (net.Conn, error) {
	network, addr, err := uriToAddr(uri)
	if err != nil {
		return nil, err
	}
	return net.Dial(network, addr)
}

func uriToAddr(uri string) (network, address string, err error) {
	if len(uri) == 0 {
		err = fmt.Errorf("Empty uri")
		return
	}

	parts := strings.SplitN(uri, "://", 2)
	switch len(parts) {
	case 1:
		address = parts[0]

		// FIXME: bad heuristic, a path can contain a ':' even if unlikely
		if strings.Contains(address, ":") {
			network = "tcp"
		} else {
			network = "unix"
		}

	case 2:
		network, address = parts[0], parts[1]
	default:
		err = fmt.Errorf("BUG")
	}
	return
}

// Internal interface implemented by net.TCPListener and net.UDPListener
type filer interface {
	File() (*os.File, error)
}

var (
	_ filer = (*net.TCPListener)(nil)
	_ filer = (*net.UDPConn)(nil)
)
