/*
Grouper implements process orcestration.  Runners are organized into groups,
which are then organized into an execution tree.  If you have modeled your subsystems
as ifrit runners, startup and shutdown of your entire application can now
be controlled.

Grouper provides four strategies for system startup: three StaticGroup
strategies, and one DynamicGroup.  Each static group strategy takes a
list of members, and starts the members in the following manner:

  - Parallel: all processes are started simultaneously.
  - Ordered:  the next process is started when the previous is ready.
  - Serial:   the next process is started when the previous exits.

The DynamicGroup allows up to N processes to be run concurrently. The dynamic
group runs indefinitely until it is closed or signaled. A dynamic group has the
following properties:

  - A dynamic group allows Members to be inserted until it is closed.
  - A dynamic group can be manually closed via it's client.
  - A dynamic group is automatically closed once it is signaled.
  - Once a dynamic group is closed, it acts like a static group.

Groups can optionally be configured with a termination signal, and all groups
have the same signaling and shutdown properties:

  - The group propogates all received signals to all running members.
  - If a member exits before being signaled, the group propogates the
    termination signal.  A nil termination signal is not propogated.

All groups provide a client.  The DynamicClient interface embeds the
StaticClient interface.
*/
package grouper
