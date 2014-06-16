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

`crankx` multiplexes `crankctl` calls over a prefix.

The prefix is prepended with `/var/crank/` unless it start with a `.` or `/`

`crankx` will then invoke `crankctl` for each `$prefix-*.ctl` file.

Example:

    crank -name api-8080 -addr :8080 &
    crank -name api-8081 -addr :8081 &
    # Starts api.rb on both crank processes in turn:
    crankx api run api.rb -wait

SEE ALSO
--------

crank(1), crankctl(1)
