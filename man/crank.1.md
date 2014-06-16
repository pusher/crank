CRANK 1 "APRIL 2014" Crank "User Manuals"
=========================================

NAME
----

crank - restart your processes, slowly

SYNOPSIS
--------

`crank` [opts]

DESCRIPTION
-----------

Crank's main goal is to handle restarts for servers who handle long-lived TCP
socket connections. Traditional process managers tend to be rather brusque and
kill the server. We want to be able to let the client an opportunity to
reconnect somewhere else. We also want to load the new version and wait until
it tells us it's ready and thus proove it's working before shutting down the
old process.

Because `crank` exits when all the child processes are gone, you should run it
under a system-level supervisor like upstart or systemd that handles restarts.

Processes run under crank needs to be adapted to benefit from all the features
than crank provides. See the "PROCESS SIDE" section for more details.

OPTIONS
-------

Note that valid addr, conf and sock values are necessary for crank to run.

`-bind` *net-uri*
  A port or path on which to bind. This socket is not used directly by crank
  but passed onto the child process using the systemd LISTEN_FDS convention.
  Note that unlike systemd we don't pass the LISTEN_PID environment variable
  (due to a limitation in the go fork/exec model)

`-conf` *config-file*
  A path where to store the last successful run command. This path needs to be
  writeable by crank and should probably be something like
  /var/crank/something.conf

`-ctl` *net-uri*
  Path or address of the control socket. This socket exposes an rcp interface
  which is consumed by the `crankctl` command-line.

`-prefix` *path*
  Sets the crank runtime directory. Defaults to `/var/crank`.

`-name` *process-name*
  If passed, it sets the `-conf` and `-ctl` arguments to
  a `$prefix/$name.$type` default (unless those are also passed).

*net-uri* format: an address can be of the following forms:

* `<path>` (no : character allowed)
* `[host]:<port>`
* `fd://<fd_num>`
* `tcp[46]://[host]:<port>`
* `unix[packet]://<path>`

PROCESS SIDE
------------

A process is responsible to start and stop gracefully.

If the process sees a LISTEN_FDS environment variable it is supposed to use
fd:3 as the accepting socket instead of binding it's own. Note that we don't
use the systemd LISTEN_PID because of go's fork/exec limitation.

If the process sees a NOTIFY_FD environment variable it is supposed to send
a "READY=1" datagram on it once it's ready to accept new client connection.

If the process receives a SIGTERM signal it is supposed to stop accepting new
connections and stop gracefully or not the existing ones. Crank will
forcefully terminate the process after a configured period.

ENVIRONMENT
-----------

`CRANK_BIND`, `CRANK_CONF`, `CRANK_CTL`, `CRANK_NAME`
  If non-null it defines the default argument of their corresponding flag.

FILES
-----

The config file contains the serialization of config of the last
successfully-started process. In that sense it should not belong in /etc.

BUGS
----

Report bugs and ideas on the github project's issue tracker.
https://github.com/pusher/crank/issues/

AUTHOR
------

Martyn Loughran <martyn@mloughran.com>
zimbatm <zimbatm@zimbatm.com>
Paweł Ledwoń <pawel@pusher.com>

SEE ALSO
--------

crankctl(1), crankx(1), sd-daemon(3)
