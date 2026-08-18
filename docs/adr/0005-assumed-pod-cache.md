# ADR 0005: In-memory assumed-pod cache to close the double-bind race

## Status
Accepted

## Context
ADR 0003 flagged that FitsNode's capacity check relies on a fresh List call,
which doesn't see pods bound moments earlier but not yet reflected by the
informer/API — risking two pods overcommitting the same capacity.

## Decision
Add an in-process AssumeCache: immediately after a successful bind, record
the pod against its node in memory. Merge assumed pods into capacity
calculations alongside the real pod list. Entries clear on informer
confirmation or a 30s TTL, whichever comes first.

## Consequences
- Closes the race for the common case (single scheduler replica).
- Does NOT address multi-replica double-binding — if we ever run more than
  one Moira instance without leader election, both replicas have independent
  caches and can still race. This is fine today (single replica), but
  leader election (Phase 1 item 8) becomes a hard requirement before running
  multiple replicas, not just a nice-to-have.
- The watch's server-side nodeName="" filter means Forget rarely fires from
  the confirmation path in practice; TTL is the real backstop. Flagged as a
  known simplification, not silently assumed away.