Crank - restart your servers, slowly
====================================

[TODO: build badge here]()

Because we're more modern than horses.

Crank's main goal is to handle restarts for servers who handle long-lived TCP
socket connections. Traditional process managers tend to be rather brusque and
kill the server. We want to be able to let the client an opportunity to
reconnect somewhere else. We also want to load the new version and wait until
it tells us it's ready and thus proove it's working before shutting down the
old process.

For example you have 6 processes on a host, all accepting on different ports.
You want to be able to restart them one by one, ideally with none loosing
child connections.

Cranks's specific goals are:
* Be able to change the command-line between restart
* Be able to start a new process, and only close the old one if it
  successfully started.
* Be able to orchestrate a reload between multiple crank processes in a
  rolling fashion.

Also, we don't really care about OS compatibility other than Linux (but OSX is
nice to have).

How it works
------------

Crank is designed to site between your process supervisor of choice and your
application.

In the init script or by hand, run `crank -bind :8080 -name $service_name`.
Crank will bind the 0.0.0.0:8080 port and do nothing (unless a process has
successfully been started in the past).

Independently or in a post-start script run
`crankctl run -name $service_name -start 10 -stop 10000 /path/to/app` to
actually start the app. `crankctl` forwards that command to crank, crank
starts the service. If everything went fine crank will record the
configuration.

During start, crank passes the bound socket to the application using the
LISTEN_FDS=1 environment variable. The app is then supposed to pick the FD3
and use it to listen to incoming connections. When the app is ready, it's
supposed to send a "READY=1" message to the LISTEN_FD. Crank knows the app is
ready and sends a SIGTERM to the old current process. That way your current
process is only terminated if the deploy was successful.

Processes' responsability
-------------------------

A process must bind on a passed file-descriptor if the LISTEN_FDS environment
variable is given.

A process must send a "READY=1" datagram to the NOTIFY_FD passed as an
environment variable.

Development
-----------

A working Go compiler and ruby environment is necessary. Run `make` to compile
new versions of crank and `rake` for new versions of the man pages.

Install
-------

For now crank is a source only distribution.
Run `make install PREFIX=/path/to/target` to install it where you want. PREFIX
targets to /usr/local if omitted.

