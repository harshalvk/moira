# ADR 0007: Single-strategy scoring selection via Config, not multi-plugin scoring

## Status
Accepted

## Context
Roadmap called for pluggable spread vs. bin-packing scoring. Registering
both NodeResourcesLeastAllocated and NodeResourcesMostAllocated
simultaneously would produce a weighted sum where they cancel toward a
constant — not a meaningful hybrid.

## Decision
Add a Config.Strategy field (spread|pack); scheduler.New registers exactly
one node-resources score plugin based on it. Default is spread, preserving
existing behavior for anyone upgrading from Step 6.

## Consequences
- scheduler.New's signature changed: now returns (*Scheduler, error) since
  invalid config must fail construction, not silently misbehave.
- Strategy is cluster-wide (one running scheduler instance = one strategy),
  not per-pod/per-namespace. Per-workload strategy selection would need a
  PriorityClass-like mechanism — out of scope until a real use case
  demands it.
- Configured via MOIRA_STRATEGY env var for now; revisit if/when more
  tunables accumulate and a proper flags/config-file layer is warranted.