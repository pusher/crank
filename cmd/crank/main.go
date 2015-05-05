package main

import (
	"flag"
	"github.com/pusher/crank/src/crank"
	"github.com/pusher/crank/src/netutil"
	"log"
	"net"
	"os"
	"syscall"
)

var (
	bind    string
	conf    string
	ctl     string
	prefix  string
	name    string
	version bool

	build string
)

func init() {
	flag.StringVar(&bind, "bind", os.Getenv("CRANK_BIND"), "external address to bind (e.g. 'tcp://:80')")
	flag.StringVar(&conf, "conf", os.Getenv("CRANK_CONF"), "path to the process config file")
	flag.StringVar(&ctl, "ctl", os.Getenv("CRANK_CTL"), "rpc socket address")
	flag.StringVar(&prefix, "prefix", crank.Prefix(os.Getenv("CRANK_PREFIX")), "crank runtime directory")
	flag.StringVar(&name, "name", os.Getenv("CRANK_NAME"), "crank process name. Used to infer -conf and -ctl if specified.")
	flag.BoolVar(&version, "version", false, "show version")
}

func main() {
	flag.Parse()

	if version {
		log.Println(crank.GetInfo(build))
		return
	}

	conf = crank.DefaultConf(conf, prefix, name)
	ctl = crank.DefaultCtl(ctl, prefix, name)

	if bind == "" {
		log.Fatal("Missing required flag: bind")
	}
	if ctl == "" {
		log.Fatal("Missing required flag: ctl or name")
	}
	if conf == "" {
		log.Fatal("Missing required flag: conf or name")
	}

	socket, err := netutil.BindURI(bind)
	if err != nil {
		log.Fatal("bind socket failed: ", err)
	}

	// Make sure the path is writeable
	f, err := os.OpenFile(conf, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal("Config file not writeable: ", err)
	}
	f.Close()

	rpcFile, err := netutil.BindURI(ctl)
	if err != nil {
		log.Fatal("ctl socket failed: ", err)
	}
	rpcListener, err := net.FileListener(rpcFile)
	if err != nil {
		log.Fatal("BUG(rpcListener) : ", err)
	}
	rpcFile.Close()
	rpcListener = netutil.UnlinkListener(rpcListener)

	manager := crank.NewManager(build, name, conf, socket)
	go onSignal(manager.Reload, syscall.SIGHUP)
	go onSignal(manager.Shutdown, syscall.SIGTERM, syscall.SIGINT)

	rpc := crank.NewRPCServer(manager)
	go rpc.Accept(rpcListener)

	manager.Run() // Blocking

	rpcListener.Close()

	log.Println("Bye!")
}
