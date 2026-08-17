#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="moira-dev"

if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  if ! kubectl cluster-info --context "kind-${CLUSTER_NAME}" &>/dev/null; then
    echo "cluster '${CLUSTER_NAME}' exists but is unreachable (common after WSL2/Docker restart)"
    echo "recreating..."
    kind delete cluster --name "${CLUSTER_NAME}"
  else
    echo "cluster '${CLUSTER_NAME}' already exists and is reachable"
    exit 0
  fi
fi

cat <<EOF | kind create cluster --name "${CLUSTER_NAME}" --config -
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
  - role: worker
EOF

echo "cluster '${CLUSTER_NAME}' ready — kubectl context set automatically"
kubectl cluster-info --context "kind-${CLUSTER_NAME}"
