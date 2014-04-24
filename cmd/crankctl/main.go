package main

import (
	"../../pkg/crank"
	"../../pkg/netutil"
	"flag"
	"fmt"
	"net/rpc"
	"os"
)

type Command func(*rpc.Client) error
type CommandSetup func(*flag.FlagSet) Command

var (
	commands map[string]CommandSetup
	flags    *flag.FlagSet
	name     string
	sock     string
)

func init() {
	commands = make(map[string]CommandSetup)
	commands["info"] = Info
	commands["kill"] = Kill
	commands["ps"] = Ps
	commands["run"] = Run

	flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [opts] <command> [command opts]\n\nOptions:\n", os.Args[0])
		flags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		for name := range commands {
			fmt.Fprintf(os.Stderr, "  %s\n", name)
		}
	}
	defaultFlags(flags)
}

func defaultFlags(flagSet *flag.FlagSet) {
	flagSet.StringVar(&sock, "sock", sock, "path to control socket")
	flagSet.StringVar(&name, "name", name, "crank process name. Used to infer -sock if specified.")
}

func usageError(reason string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+reason+"\n\n", args...)
	flags.Usage()
	os.Exit(1)
}

func fail(reason string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+reason+"\n", args...)
	os.Exit(1)
}

func main() {
	var err error

	if err = flags.Parse(os.Args[1:]); err != nil {
		usageError("%s", err)
	}

	command := flags.Arg(0)

	if command == "" {
		usageError("command missing")
	}

	cmdSetup, ok := commands[command]
	if !ok {
		usageError("unknown command %s", command)
	}

	flagSet := flag.NewFlagSet(os.Args[0]+" "+command, flag.ExitOnError)
	defaultFlags(flagSet)

	cmd := cmdSetup(flagSet)

	if err = flagSet.Parse(flags.Args()[1:]); err != nil {
		usageError("%s", err)
	}

	sock = crank.DefaultSock(sock, name)
	conn, err := netutil.DialURI(sock)
	if err != nil {
		fail("couldn't connect: %s", err)
	}
	client := rpc.NewClient(conn)

	if err = cmd(client); err != nil {
		fail("command failed: %v", err)
	}
}

func Run(flag *flag.FlagSet) Command {
	query := crank.StartQuery{}
	flag.IntVar(&query.StopTimeout, "stop", -1, "Stop timeout in seconds")
	flag.IntVar(&query.StartTimeout, "start", -1, "Start timeout in seconds")
	//flag.BoolVar(&query.Wait, "wait", false, "Wait for a result")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s run [opts] -- [command ...args]:\n", os.Args[0])
		flag.PrintDefaults()
	}

	return func(client *rpc.Client) (err error) {
		var reply crank.StartReply

		// Command and args are passed after
		if flag.NArg() > 0 {
			query.Command = flag.Args()
		}

		if err = client.Call("crank.Run", &query, &reply); err != nil {
			return
		}

		return
	}
}

func Info(flag *flag.FlagSet) Command {
	query := crank.InfoQuery{}

	return func(client *rpc.Client) (err error) {
		var reply crank.InfoReply

		if err = client.Call("crank.Info", &query, &reply); err != nil {
			return
		}

		// TODO: version, ...
		fmt.Println("goroutines:", reply.NumGoroutine)

		return
	}
}

func Ps(flag *flag.FlagSet) Command {
	query := crank.PsQuery{}
	processQueryFlags(&query.ProcessQuery, flag)

	return func(client *rpc.Client) (err error) {
		var reply crank.PsReply

		if err = client.Call("crank.Ps", &query, &reply); err != nil {
			return
		}

		for _, pi := range reply.PS {
			fmt.Println(pi)
		}

		return
	}
}

func Kill(flag *flag.FlagSet) Command {
	query := crank.KillQuery{}
	processQueryFlags(&query.ProcessQuery, flag)
	flag.StringVar(&query.Signal, "signal", "SIGTERM", "signal to send to the processes")
	flag.BoolVar(&query.Wait, "wait", false, "wait for the target processes to exit")

	return func(client *rpc.Client) (err error) {
		var reply crank.KillReply

		return client.Call("crank.Kill", &query, &reply)
	}
}

func processQueryFlags(query *crank.ProcessQuery, flag *flag.FlagSet) {
	flag.BoolVar(&query.Starting, "starting", false, "lists the starting process")
	flag.BoolVar(&query.Ready, "ready", false, "lists the ready process")
	flag.BoolVar(&query.Stopping, "stoppping", false, "lists all processes shutting down")
	flag.IntVar(&query.Pid, "pid", 0, "filters to only include that pid")
}
