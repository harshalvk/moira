# ADR 0004: Required constraints are filters, preferred constraints are scoring

## Status
Accepted

## Context
Both taints (NoSchedule vs PreferNoSchedule) and node affinity (required vs
preferred) have this same required/preferred split. Needed a consistent rule
for where each lives in the pipeline.

## Decision
Anything "required"/"NoSchedule"/"NoExecute" is a hard filter (excludes the
node from the candidate set entirely). Anything "preferred" is deferred to
Phase 1 scoring — filtering never considers it.

## Consequences
- Filtering functions stay boolean and simple.
- Preferred-affinity and PreferNoSchedule taints currently have NO effect
  on scheduling at all (since scoring doesn't exist yet) — pods with only
  preferred constraints will be placed as if they had none. This is
  expected and temporary; resolved when Phase 1 scoring lands.