Crank - restart your servers, slowly
====================================

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

Crank design
------------

Crank's design is to hold onto a specific port by opening it and then passing
it onto child processes trough the fork/exec chain.

Resource management
-------------------

The `crankctl` command-line exposes RSS usage trough the `ps` sub-command.
This allows to make tooling that monitors that value and issues a restart if
necessary.

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



