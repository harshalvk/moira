# Moira

A Kubernetes scheduler built from first principles — starting as a naive
random-placement loop and progressing toward gang scheduling and tenant-fair
queueing, implemented as a plugin-based scheduler (Filter/Score extension
points) structurally mirroring the real scheduler-framework, so a future
migration to `k8s.io/kubernetes/pkg/scheduler/framework` is a mechanical
swap rather than a rewrite (see ADR 0006).

## Why

Existing schedulers each leave something on the table:
- **kube-scheduler (default)** — no first-class gang scheduling, weak
  batch/ML workload support, generic non-workload-aware preemption.
- **Volcano** — strong batch/gang scheduling, but heavy to operate and
  tightly coupled to its own CRDs rather than being a clean drop-in.
- **Apache YuniKorn** — good multi-tenant fairness, but requires app-level
  integration and has a smaller plugin ecosystem.
- **Kueue** — job queueing on top of a scheduler, not a scheduler itself —
  doesn't solve node-level placement.

Moira's target gap: framework-native placement with gang scheduling and
tenant fairness built in from the start, without Volcano's operational
weight.

See `docs/adr/` for the full decision history and current known gaps.

## Features

- Watches pods with `spec.schedulerName: moira` and binds them to a node
- **Filter plugins** (hard constraints — reject a node outright):
  - `NodeResourcesFit` — CPU/memory requests vs. node allocatable
  - `TaintToleration` — NoSchedule/NoExecute taint matching
    (`PreferNoSchedule` is advisory-only, deferred to scoring — ADR 0004)
  - `NodeAffinity` — `requiredDuringSchedulingIgnoredDuringExecution`
    node selector terms (`In`/`NotIn`/`Exists`/`DoesNotExist`)
- **Score plugins** (rank feasible nodes, weighted sum):
  - `NodeResourcesLeastAllocated` — spread strategy, favors emptier nodes
- **Assumed-pod cache** — closes the race between binding a pod and the
  informer confirming it, so back-to-back scheduling decisions don't
  overcommit the same node (ADR 0005)
- Structured JSON logging, graceful shutdown on SIGINT/SIGTERM

## Project structure
```
moira/
├── cmd/
│ └── moira/
│ └── main.go # entrypoint: kubeconfig client, wiring, shutdown
├── internal/
│ ├── framework/ # plugin interfaces + registry (not upstream k8s — ADR 0006)
│ │ ├── types.go # FilterPlugin/ScorePlugin, Status, NodeInfo
│ │ ├── registry.go # RunFilterPlugins / RunScorePlugins
│ │ └── registry_test.go
│ ├── plugins/
│ │ ├── noderesourcesfit/ # Filter: CPU/mem fit
│ │ ├── tainttoleration/ # Filter: taints/tolerations
│ │ ├── nodeaffinity/ # Filter: required node affinity
│ │ └── noderesourcesleastallocated/ # Score: spread strategy
│ ├── scheduler/
│ │ ├── scheduler.go # watch->decide->bind loop, plugin wiring
│ │ └── cache.go # AssumeCache (double-bind race fix)
│ └── version/
│ └── version.go
├── hack/
│ ├── kind-cluster.sh # spin up local 3-worker kind cluster
│ └── kind-cluster-down.sh
├── docs/
│ └── adr/ # architecture decision records, 0001-0006
├── .devcontainer/ # zero-setup onboarding
├── .github/workflows/ci.yml # build, test, lint
├── .golangci.yml
├── .air.toml # hot-reload config for make dev
├── lefthook.yml # pre-commit fmt/vet/lint, pre-push test
└── Makefile
```


## Quickstart

```bash
git clone https://github.com/harshalvk/moira && cd moira
make setup      # installs air, golangci-lint, lefthook + git hooks
make kind-up    # local 3-worker kind cluster (kind-moira-dev context)
make dev        # hot-reloads moira on file save
```

Deploy any pod with `spec.schedulerName: moira` and watch it land on the
emptiest feasible node.

