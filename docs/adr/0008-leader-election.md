# ADR 0008: Leader election via client-go Lease, not a custom mechanism

## Status
Accepted

## Context
ADR 0005 and ADR 0007 both flagged that multiple Moira replicas without
coordination can race (double-binding, and independently-configured
strategy divergence). Needed a standard HA mechanism before recommending
>1 replica.

## Decision
Use client-go's tools/leaderelection package against a coordination.k8s.io
Lease object — the same mechanism kube-scheduler, kube-controller-manager,
and most controllers use. NewLeaderElector (not RunOrDie) so setup failures
return an error instead of panicking.

## Consequences
- Only the leader replica runs the scheduling loop; standbys block until
  they acquire the lease (e.g., on leader crash/pod eviction).
- OnStartedLeading's context is cancelled the instant leadership is lost —
  s.Run(leadCtx) must (and does) stop promptly on ctx cancellation, since
  continuing to schedule after losing leadership would reintroduce the
  exact race this step closes.
- Config.Enabled=false path exists for local dev only (make dev) — a real
  deployment should never disable this with >1 replica.
- New gap opened, not yet closed: RBAC manifests for the scheduler's core
  permissions (node/pod list/watch/bind) don't exist yet, only the
  leader-election-specific Role added here. A full deploy/ manifest set is
  needed before real in-cluster deployment — tracked as follow-up, not
  silently assumed to exist.