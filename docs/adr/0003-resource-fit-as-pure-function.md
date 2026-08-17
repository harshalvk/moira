# ADR 0003: Resource-fit filtering as a pure, cache-free function

## Status
Accepted

## Context
Filtering logic needs to grow substantially (taints, affinity, gang
constraints) and needs to be fast to unit-test without spinning up fake
clientsets for every case.

## Decision
`FitsNode` takes pod, node, and existing pods as plain arguments and
returns a bool — no cluster access inside the function itself.

## Consequences
- Trivial, fast unit tests (see fit_test.go) with no fake clientset.
- Known gap: doesn't account for pods "assumed" scheduled but not yet
  visible via List (binding latency race). Acceptable for Phase 0; must
  be solved with an internal assumed-pod cache before Phase 1's scoring
  work, since scoring under contention makes the race more likely to bite.