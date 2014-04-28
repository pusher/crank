CRANKX 1 "APRIL 2014" Crank "User Manuals"
==========================================

NAME
----

crankx - crank multi-control client

SYNOPSIS
--------

`crankx` `<prefix>` [crankctl opts]

DESCRIPTION
-----------

`crankx` multiplexes `crankctl` calls over a prefix. It's useful only for
crank processes who have their sock files under /var/run/crank/$prefix-*.sock

Example:

    crank -name api-8080 -addr :8080 &
    crank -name api-8081 -addr :8081 &
    # Starts api.rb on both crank processes in turn:
    crankx api run api.rb -wait

SEE ALSO
--------

crank(1), crankctl(1)
