# ADR 0009: Prometheus metrics via a dedicated, non-leader-gated HTTP server

## Status
Accepted

## Context
Roadmap item 9 called for metrics. Also needed: something concrete for the
upcoming kind-based integration tests to assert against beyond "the pod
landed somewhere."

## Decision
- prometheus/client_golang with promauto for registration (panics loudly
  on duplicate metric names at startup, not silent misregistration later).
- Four metrics: scheduling latency (histogram), scheduling attempts by
  outcome (counter), per-plugin duration by extension point (histogram),
  leader status (gauge).
- Metrics/health server runs independently of leader election — every
  replica exposes /metrics and /healthz regardless of leadership status.

## Consequences
- A standby replica being unreachable is now independently observable,
  not masked by only the leader ever running an HTTP server.
- Plugin-duration instrumentation lives in the Registry, not per-plugin —
  new plugins get timing for free without remembering to add it themselves.
- No metrics on the AssumeCache (size, eviction count) yet — worth adding
  once gang scheduling (Phase 2) makes cache behavior more load-bearing.