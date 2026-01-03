#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-}"          # down|up
NODE="${2:-}"            # node name
ROOK_OPERATOR_NAMESPACE="${ROOK_OPERATOR_NAMESPACE:-rook-ceph}"
ROOK_CLUSTER_NAMESPACE="${ROOK_CLUSTER_NAMESPACE:-rook-ceph}"
STATE_FILE="${STATE_FILE:-./rook-node-maint-${NODE}.tsv}"

# Prefixes to consider "node-scoped enough" to stop on a single node
# (OSDs are typically node-pinned; crashcollector/exporter often are too)
PREFIXES_REGEX='^(rook-ceph-osd|rook-ceph-mon|rook-ceph-exporter|rook-ceph-crashcollector)'

usage() {
  echo "Usage: $0 down|up <node-name>"
  echo "Env vars: ROOK_OPERATOR_NAMESPACE, ROOK_CLUSTER_NAMESPACE, STATE_FILE"
}

require() { command -v "$1" >/dev/null 2>&1 || { echo "Missing required command: $1" >&2; exit 1; }; }

wait_deploy_readyreplicas_empty() {
  local ns="$1" name="$2"
  # Doc uses checking readyReplicas becomes "" when scaled to 0 :contentReference[oaicite:4]{index=4}
  while [[ "$(kubectl -n "$ns" get deploy "$name" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || true)" != "" ]]; do
    sleep 5
  done
}

wait_deploy_replicas_eq() {
  local ns="$1" name="$2" desired="$3"
  while [[ "$(kubectl -n "$ns" get deploy "$name" -o jsonpath='{.status.replicas}' 2>/dev/null || echo 0)" != "$desired" ]]; do
    sleep 5
  done
}

get_target_deployments_on_node() {
  # Find pods on NODE, map Pod -> owning Deployment (via ReplicaSet), return unique deployment names
  kubectl -n "$ROOK_CLUSTER_NAMESPACE" get pods --field-selector "spec.nodeName=${NODE}" -o json \
  | jq -r '
      .items[]
      | .metadata.ownerReferences[]?
      | select(.kind=="ReplicaSet")
      | .name
    ' \
  | sort -u \
  | while read -r rs; do
      kubectl -n "$ROOK_CLUSTER_NAMESPACE" get rs "$rs" -o json \
      | jq -r '.metadata.ownerReferences[]? | select(.kind=="Deployment") | .name'
    done \
  | grep -E "$PREFIXES_REGEX" \
  | sort -u
}

ceph_cmd() {
  # Requires rook-ceph-tools deployment from the docs :contentReference[oaicite:5]{index=5}
  kubectl -n "$ROOK_CLUSTER_NAMESPACE" exec deploy/rook-ceph-tools -- ceph "$@"
}

main() {
  require kubectl
  require jq

  if [[ -z "$ACTION" || -z "$NODE" ]]; then usage; exit 1; fi
  if [[ "$ACTION" != "down" && "$ACTION" != "up" ]]; then usage; exit 1; fi

  if [[ "$ACTION" == "down" ]]; then
    echo "==> Cordoning node: $NODE"
    kubectl cordon "$NODE"

    echo "==> Setting Ceph flag: noout"
    ceph_cmd osd set noout

    echo "==> Scaling down rook operator (prevents reconciliation restarting what we stop)"
    kubectl -n "$ROOK_OPERATOR_NAMESPACE" scale deployment rook-ceph-operator --replicas=0

    echo "==> Discovering target deployments with pods on node: $NODE"
    mapfile -t deps < <(get_target_deployments_on_node)
    if [[ "${#deps[@]}" -eq 0 ]]; then
      echo "No matching deployments found on node $NODE (regex: $PREFIXES_REGEX)"
    fi

    : > "$STATE_FILE"
    for d in "${deps[@]}"; do
      cur="$(kubectl -n "$ROOK_CLUSTER_NAMESPACE" get deploy "$d" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo 0)"
      echo -e "Deployment\t${ROOK_CLUSTER_NAMESPACE}\t${d}\t${cur}" >> "$STATE_FILE"

      echo "==> Scaling ${ROOK_CLUSTER_NAMESPACE}/deploy/${d}: ${cur} -> 0"
      kubectl -n "$ROOK_CLUSTER_NAMESPACE" scale deployment "$d" --replicas=0
      wait_deploy_readyreplicas_empty "$ROOK_CLUSTER_NAMESPACE" "$d"
    done

    echo
    echo "==> Saved state to $STATE_FILE"
    echo "==> Now you can reboot/maintain node $NODE."
    echo "    When done, run: $0 up $NODE"

  else # up
    if [[ ! -f "$STATE_FILE" ]]; then
      echo "State file not found: $STATE_FILE" >&2
      echo "Nothing to restore." >&2
      exit 1
    fi

    echo "==> Restoring deployments from $STATE_FILE"
    while IFS=$'\t' read -r kind ns name replicas; do
      [[ "$kind" == "Deployment" ]] || continue
      echo "==> Scaling $ns/deploy/$name -> $replicas"
      kubectl -n "$ns" scale deployment "$name" --replicas="$replicas"
      wait_deploy_replicas_eq "$ns" "$name" "$replicas"
    done < "$STATE_FILE"

    echo "==> Scaling rook operator back up"
    kubectl -n "$ROOK_OPERATOR_NAMESPACE" scale deployment rook-ceph-operator --replicas=1

    echo "==> Unsetting Ceph flag: noout"
    ceph_cmd osd unset noout

    echo "==> Uncordoning node: $NODE"
    kubectl uncordon "$NODE"

    echo "==> Done."
  fi
}

main "$@"

