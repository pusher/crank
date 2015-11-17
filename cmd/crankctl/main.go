package main

import (
	"flag"
	"fmt"
	"net/rpc"
	"os"

	"github.com/pusher/crank/src/crank"
	"github.com/pusher/crank/src/netutil"
)

type Command func(*rpc.Client) error
type CommandSetup func(*flag.FlagSet) Command

type ExitError int

func (e ExitError) Error() string {
	return fmt.Sprintf("exited with %d", e)
}

var (
	commands map[string]CommandSetup
	flags    *flag.FlagSet
	ctl      string = os.Getenv("CRANK_CTL")
	prefix   string = crank.Prefix(os.Getenv("CRANK_PREFIX"))
	name     string = os.Getenv("CRANK_NAME")
	version  bool

	build string
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
	flags.BoolVar(&version, "version", false, "show version")
}

func defaultFlags(flagSet *flag.FlagSet) {
	flagSet.StringVar(&ctl, "ctl", ctl, "path or address of the control socket")
	flagSet.StringVar(&prefix, "prefix", prefix, "crank runtime directory")
	flagSet.StringVar(&name, "name", name, "crank process name. Used to infer -ctl if specified.")
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

	if version {
		fmt.Println(crank.GetInfo(build))
		return
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

	ctl = crank.DefaultCtl(ctl, prefix, name)
	conn, err := netutil.DialURI(ctl)
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
	flag.IntVar(&query.Pid, "pid", 0, "Only if the current pid matches")
	flag.BoolVar(&query.Wait, "wait", false, "Wait for a result")
	flag.StringVar(&query.Cwd, "cwd", "", "Working directory")

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
			fmt.Println("Failed to start:", err)
			return
		}
		if reply.Code > 0 {
			fmt.Println("Exited with code:", reply.Code)
			return ExitError(reply.Code)
		}

		fmt.Println("Started successfully")
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

		fmt.Printf("crankctl\n-------\n%s\n\n", crank.GetInfo(build))
		fmt.Printf("crank\n-----\n%s\n", reply.Info)

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
