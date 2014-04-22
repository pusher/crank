package main

import (
	"../../pkg/crank"
	"flag"
	"fmt"
	"net/rpc"
	"os"
)

type Command func(*rpc.Client) error
type CommandSetup func(*flag.FlagSet) Command

var flags *flag.FlagSet
var run string
var commands map[string]CommandSetup

func init() {
	commands = make(map[string]CommandSetup)
	commands["ps"] = Ps
	commands["kill"] = Kill

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
	flagSet.StringVar(&run, "run", run, "path to control socket")
}

func fail(reason string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, reason, args...)
	flags.Usage()
	os.Exit(1)
}

func main() {
	var err error

	if err = flags.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	command := flags.Arg(0)

	if command == "" {
		fail("command missing\n")
	}

	cmdSetup, ok := commands[command]
	if !ok {
		fail("unknown command %s\n", command)
	}

	flagSet := flag.NewFlagSet(os.Args[0]+" "+command, flag.ExitOnError)
	defaultFlags(flagSet)

	cmd := cmdSetup(flagSet)

	if err = flagSet.Parse(flags.Args()[1:]); err != nil {
		fail("oops: %s\n", err)
	}

	client, err := rpc.Dial("unix", run)
	if err != nil {
		fail("couldn't connect: %s\n", err)
	}

	if err = cmd(client); err != nil {
		fail("command failed: %v\n", err)
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
			fmt.Printf("%d %s\n", pi.Pid, pi.State)
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
	flag.BoolVar(&query.Start, "start", false, "lists the starting process")
	flag.BoolVar(&query.Current, "current", false, "lists the current process")
	flag.BoolVar(&query.Shutdown, "shutdown", false, "lists all processes shutting down")
	flag.IntVar(&query.Pid, "pid", 0, "filters to only include that pid")
}
