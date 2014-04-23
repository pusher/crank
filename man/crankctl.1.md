CRANKCTL 1 "APRIL 2014" Crank "User Manuals"
============================================

NAME
----

crankctl - crank control client

SYNOPSIS
--------

`crankctl` `<command>` [opts]

DESCRIPTION
-----------

`crankctl` allows to send commands to `crank` trough it's control socket.

OPTIONS
-------

Global options:

`-name` *process-name*
  If passed, it sets the -sock arguments to
  a `/var/run/crank/$name.sock` default.

`-sock` *path*
  A path on which to connect. This should point to an existing unix socket
  controlled by crank.

COMMANDS
--------

* `crankctl run [opts] -- [command ...args]`

Used to start a new process. Once ready, crank terminates the old process. If
the startup fails, crank leaves the old process running and untouched.

`-start SEC`
  Sets the start timeout of the process in seconds.

`-stop SEC`
  Sets the stop timeout of the process in seconds.

`-wait`
  Waits for either the process to be ready or to fail. If the new process has
  failed, crankctl exits with an exit status of 1.

`command ...args`
  Gives the command and args to run. If unspecified, the previous successful
  command is used.

* `crankctl ps [opts]`

Displays the status of running processes. If no argument is passed, all
processes are listed.

`-starting`
  Selects all starting processes (should only be one)

`-ready`
  Selects all ready processes (should only be one)

`-stoppping`
  Selects all stoppping processes.

`-pid PID`
  Selects a specific PID from the exisiting set. This flag is a AND filter
  unlike the other ones.

* `crankctl kill [opts]`

Sends a signal to the target processes. If no argument is passes, no processes
are signaled.

`-signal SIGNAME`
  Provides the type of signal to send. If no signal is passed, SIGTERM is the
  default. Signals can be prefixed with "SIG" or not. Eg: SIGINT or INT

`-starting`
  Selects all starting processes (should only be one)

`-ready`
  Selects all ready processes (should only be one)

`-stoppping`
  Selects all stoppping processes.

`-pid PID`
  Selects a specific PID from the exisiting set. This flag is a AND filter
  unlike the other ones.

ENVIRONMENT
-----------

`CRANK_NAME`, `CRANK_SOCK`
  If non-null it defines the default argument of their corresponding flag.

SEE ALSO
--------

crank(1)
