# ADR 0001: Language choice and project scope

## Status
Accepted

## Context
We're building a Kubernetes scheduler from scratch, progressing from a naive
placement loop to a gang-scheduling/fairness-aware scheduler comparable in
ambition to Volcano/YuniKorn, but framework-native (drop-in via the real
scheduler-framework extension points, not a fork).

Candidate languages: Go, Rust, TypeScript/Node.js.

## Decision
Go, using client-go and eventually the upstream scheduler-framework
interfaces directly.

## Rationale
- The entire k8s ecosystem (client-go, controller-runtime, kubebuilder,
  scheduler-framework) is Go-native. Building in Go means extending
  well-trodden patterns instead of reimplementing informers, work queues,
  and leader election from scratch.
- Rust (kube-rs) is viable but immature at the scheduler-framework layer —
  would mean hand-rolling infrastructure Go gets for free.
- TypeScript/Node.js has no mature watch/informer-capable k8s client and
  isn't credible for a scheduler running in real clusters. Reserved for a
  possible dashboard layer later.

## Consequences
- Locks us into Go idioms/tooling for the scheduler core.
- Reuses experience from the Kairos project (Go, distributed systems).
- A future dashboard/visualization layer, if built, will be a separate
  service (likely TS/Next.js) consuming Moira's metrics/API — not part of
  the scheduler core.