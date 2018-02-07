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

`crankctl` allows to send commands to `crank` trough it's control port.

OPTIONS
-------

Global options:

`-name` *process-name*
  If passed, it sets the `-ctl` arguments to a `$prefix/$name.ctl` default.

`-prefix` *path*
  Sets the crank runtime directory. Defaults to `/var/crank`.

`-ctl` *net-uri*
  Path or address of the control port. This should point to an existing unix
  socket controlled by crank.

COMMANDS
--------

* `crankctl run [opts] -- [command ...args]`

Used to start a new process. Once ready, crank terminates the old process. If
the startup fails, crank leaves the old process running and untouched.

`-cwd PATH`
  Directory name to run the command under.

`-start SEC`
  Sets the start timeout of the process in seconds.

`-stop SEC`
  Sets the stop timeout of the process in seconds.

`-wait`
  Waits for either the process to be ready or to fail. If the new process has
  failed, crankctl exits with an exit status of 1.

`-pid PID`
  If passed crank will only spawn a new process if the current process matches
  the pid. It's useful to avoid race conditions if multiple tools interact
  with crank at the same time.

`command ...args`
  Gives the command and args to run. If unspecified, the previous successful
  command is used.

* `crankctl info [opts]`

Returns infos on the crankctl runtime.

* `crankctl ps [opts]`

Displays the status of running processes. If no argument is passed, all
processes are listed.

`-starting`
  Selects all starting processes (should only be one)

`-ready`
  Selects all ready processes (should only be one)

`-stopping`
  Selects all stopping processes.

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

`-stopping`
  Selects all stopping processes.

`-pid PID`
  Selects a specific PID from the exisiting set. This flag is a AND filter
  unlike the other ones.

ENVIRONMENT
-----------

`CRANK_NAME`, `CRANK_CTL`
  If non-null it defines the default argument of their corresponding flag.

SEE ALSO
--------

crank(1), crankmulti(1)
