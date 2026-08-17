# ADR 0002: Developer experience tooling

## Status
Accepted

## Context
A scheduler project involves a slow feedback loop by nature (build → deploy →
watch cluster behavior). Poor DX compounds that pain and kills momentum.

## Decision
- **lefthook** over pre-commit/Husky: Go-native, single static binary, no
  Python/Node runtime dependency for a Go-only repo.
- **air** for hot reload during local iteration against kind.
- **kind** (not minikube/k3d) for local clusters: fastest startup, easiest
  multi-node simulation (needed later for gang-scheduling/multi-node tests),
  scriptable via `hack/*.sh` for one-command setup.
- **golangci-lint** with gosec + revive enabled from day one, not bolted on
  once the codebase is large enough that fixing lint debt is painful.
- **devcontainer** so a fresh clone reaches a working environment without
  manual toolchain setup.

## Consequences
- New contributors go from clone to running scheduler in 3 commands.
- Hooks enforce fmt/vet/lint at commit time, not discovered in CI minutes later.
- kind's 3-worker default config anticipates Phase 2 gang-scheduling tests,
  which need multiple schedulable nodes to be meaningful.
