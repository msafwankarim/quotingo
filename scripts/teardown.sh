#!/usr/bin/env bash
# Tears down everything created by setup.sh, in reverse order.
# Safe to re-run; each step tolerates missing resources.
#
# Usage:
#   PROJECT_ID=your-project-id ./scripts/teardown.sh

set -uo pipefail

PROJECT_ID="${PROJECT_ID:-<PROJECT_ID>}"
REGION="${REGION:-us-central1}"
ZONE="${ZONE:-us-central1-a}"
AR_REPO="${AR_REPO:-quotingo}"
CLUSTER="${CLUSTER:-quotingo-cluster}"

if [[ "$PROJECT_ID" == "<PROJECT_ID>" ]]; then
  echo "ERROR: set PROJECT_ID (e.g. PROJECT_ID=my-proj ./scripts/teardown.sh)" >&2
  exit 1
fi

echo "==> Delete CD trigger"
gcloud builds triggers delete quotingo-cd --quiet --project="$PROJECT_ID" || true

echo "==> Delete CI trigger"
gcloud builds triggers delete quotingo-ci --quiet --project="$PROJECT_ID" || true

echo "==> Delete Secret Manager secret"
gcloud secrets delete quotingo-env-pat --quiet --project="$PROJECT_ID" || true

echo "==> Delete workload (frees LoadBalancer IP promptly)"
gcloud container clusters get-credentials "$CLUSTER" \
  --zone="$ZONE" --project="$PROJECT_ID" 2>/dev/null || true
kubectl delete namespace quotingo --ignore-not-found --wait=true || true

echo "==> Delete GKE cluster"
gcloud container clusters delete "$CLUSTER" \
  --zone="$ZONE" --quiet --project="$PROJECT_ID" || true

echo "==> Delete Artifact Registry repo"
gcloud artifacts repositories delete "$AR_REPO" \
  --location="$REGION" --quiet --project="$PROJECT_ID" || true

echo ""
echo "Teardown complete. Verify no GKE cluster, no forwarding rules, no triggers remain."
