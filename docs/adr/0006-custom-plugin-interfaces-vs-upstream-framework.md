# ADR 0006: Custom Filter/Score interfaces now, upstream framework deferred

## Status
Accepted

## Context
Moira's stated differentiator (ADR 0001) is being framework-native — a
drop-in replacement using real scheduler-framework extension points, not a
fork. The literal way to do that is importing k8s.io/kubernetes/pkg/scheduler/framework
and building on cmd/kube-scheduler/app, matching how scheduler-plugins and
Volcano's framework-native plugins work.

## Decision
Define our own FilterPlugin/ScorePlugin interfaces, structurally mirroring
the real framework's shape (same Filter/Score signatures, same Status
model, same 0-100 score convention), rather than importing k8s.io/kubernetes
now.

## Rationale
- k8s.io/kubernetes isn't packaged as a clean library dependency — pulling
  it in requires replace directives across many k8s.io/* modules to avoid
  version skew, and drags in a large, fast-churning dependency surface.
- The plugin *shape* is what matters for the learning goals and for a clean
  future migration; the *dependency* can be swapped later.
- At this stage (Phase 1, first plugin pass) the interface surface will
  still change (gang scheduling, preemption in Phase 2 will likely need
  new extension points beyond Filter/Score — PreFilter, Reserve, etc.).
  Better to stabilize our own interfaces against real usage first.

## Consequences
- Moira is NOT currently a true upstream drop-in — it's structurally
  similar but implements its own Filter/Score/Registry, not the real
  framework's.
- This is a deliberate, tracked gap, not an accidental one. Revisit once
  Phase 2's extension-point needs are clearer — migrating well-tested Filter/
  Score plugins into real framework.FilterPlugin/framework.ScorePlugin
  implementations should be mostly a rename + signature adjustment at that
  point, not a redesign, because we mirrored the shape deliberately.