/*
A process model for go.

Ifrit is a small set of interfaces for composing single-purpose units of work
into larger programs. Users divide their program into single purpose units of
work, each of which implements the `Runner` interface  Each `Runner` can be
invoked to create a `Process` which can be monitored and signaled to stop.

The name Ifrit comes from a type of daemon in arabic folklore.

Ifrit ships with a standard library which contains packages for common
processes - such as http servers - alongside packages which model process
supervision and orchestration - such as process groups and restart strategies.
Composing `Runners` together to forms a larger `Runner` which invokes multiple
processes.

The advantage of small, single-responsibility processes is that they are simple,
and thus can be made reliable.  Ifrit's specifications are designed to be free
of race conditions and edge cases, allowing larger orcestrated process to also
be made reliable.  The overall effect is less code and more reliability as your
system grows with grace.
*/
package ifrit


