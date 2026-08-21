# Architecture Decision Records

This folder records the significant technical decisions made while building moira -- what was decided, why, and what alternatives were considered. The goal is that anyone (including future-me) can understand *why* the codebase looks the way it does without having to reverse-engieer it form the code alone.

## Index
| ADR | Title |
|---|---|
|[0001](0001-language-and-project-scope.md)| Language choice and project scope|
|[0002](0002-developer-experience-tooling.md)| DX tooling|
|[0003](0003-resource-fit-as-pure-function.md) |Resource-fit filtering as a pure, cache-free function|
|[0004](0004-required-vs-preferred-scheduling-constraints.md) |Required constraints are filters, preferred constraints are scoring|
|[0005](0005-assumed-pod-cache.md)|In-memory assumed-pod cache to close the double-bind race|
|[0006](0006-custom-plugin-interfaces-vs-upstream-framework.md)|Custom Filter/Score interfaces now, upstream framework deferred|
|[0007](0007-pluggable-scoring-strategy.md)|Single-strategy scoring selection via Config, not multi-plugin scoring|
|[0008](0008-leader-election.md)|Leader election via client-go Lease, not a custom mechanism|