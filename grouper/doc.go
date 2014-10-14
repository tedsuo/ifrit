/*
Grouper implements process orcestration.  Runners are organized into groups,
which are then organized into an execution tree.  If you have modeled your subsystems
as ifrit runners, startup and shutdown of your entire application can now
be controlled.

Grouper provides four strategies for system startup: three StaticGroup
strategies, and one DynamicGroup strategy.  Each StaticGroup strategy takes a
list of members, and starts the members in the following manner:

  - Parallel: all processes are started simultaneously.
  - Ordered:  the next process is started when the previous is ready.
  - Serial:   the next process is started when the previous exits.

The single DynamicGroup strategy offered is called a Pool.  A Pool allows up to
N processes to be run concurrently.   and runs until it is closed.

  - A DynamicGroup allows Members to be inserted until it is closed.
  - Once a DynamicGroup is closed, it acts like a StaticGroup.
  - A DynamicGroup is automatically closed once it is signaled.

All group strategies take a termination signal, have the same shutdown
semantics:

  - The group propogates all signals to all running members.
  - If a member exits before being signaled, the group propogates the
    termination signal.  A nil termination signal is not propogated.

All groups provide a client.  The DynamicClient interface is a superset of
the StaticClient interface.
*/
package grouper
