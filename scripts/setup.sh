#!/usr/bin/env bash
# Provisions all GCP resources needed for the quotingo GitOps pipeline.
# Not idempotent — run against a fresh project, or run teardown.sh first.
#
# Prerequisites (done manually before invoking this script):
#   1. A GCP project with billing enabled
#   2. `gcloud auth login` and `gcloud config set project <PROJECT_ID>`
#   3. Console → Cloud Build → Triggers → "Connect repository":
#        - auth the GitHub App
#        - install it on msafwankarim/quotingo and msafwankarim/quotingo-env
#   4. A fine-grained GitHub PAT with contents: read/write on quotingo-env,
#      exported as `GH_PAT` in the environment
#
# Usage:
#   export GH_PAT=ghp_xxx
#   PROJECT_ID=your-project-id ./scripts/setup.sh

set -euo pipefail

PROJECT_ID="${PROJECT_ID:-<PROJECT_ID>}"
REGION="${REGION:-us-central1}"
ZONE="${ZONE:-us-central1-a}"
AR_REPO="${AR_REPO:-quotingo}"
CLUSTER="${CLUSTER:-quotingo-cluster}"
GITHUB_OWNER="${GITHUB_OWNER:-msafwankarim}"

if [[ "$PROJECT_ID" == "<PROJECT_ID>" ]]; then
  echo "ERROR: set PROJECT_ID (e.g. PROJECT_ID=my-proj ./scripts/setup.sh)" >&2
  exit 1
fi
if [[ -z "${GH_PAT:-}" ]]; then
  echo "ERROR: export GH_PAT before running" >&2
  exit 1
fi

echo "==> 1/7: Enable required APIs"
gcloud services enable \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  container.googleapis.com \
  secretmanager.googleapis.com \
  --project="$PROJECT_ID"

echo "==> 2/7: Create Artifact Registry repository"
gcloud artifacts repositories create "$AR_REPO" \
  --repository-format=docker \
  --location="$REGION" \
  --project="$PROJECT_ID"

echo "==> 3/7: Create GKE cluster (zonal, 1 x e2-small)"
gcloud container clusters create "$CLUSTER" \
  --zone="$ZONE" \
  --num-nodes=1 \
  --machine-type=e2-small \
  --disk-size=20 \
  --release-channel=regular \
  --project="$PROJECT_ID"

echo "==> 4/7: Grant IAM roles to the default Cloud Build SA"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
CB_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"
for ROLE in \
  roles/artifactregistry.writer \
  roles/secretmanager.secretAccessor \
  roles/container.developer \
  roles/container.clusterViewer; do
    gcloud projects add-iam-policy-binding "$PROJECT_ID" \
      --member="serviceAccount:${CB_SA}" \
      --role="$ROLE" \
      --condition=None >/dev/null
done

echo "==> 5/7: Store GitHub PAT in Secret Manager"
echo -n "$GH_PAT" | gcloud secrets create quotingo-env-pat \
  --replication-policy=automatic \
  --data-file=- \
  --project="$PROJECT_ID"

echo "==> 6/7: Create CI trigger (app repo)"
gcloud builds triggers create github \
  --name=quotingo-ci \
  --repo-owner="$GITHUB_OWNER" \
  --repo-name=quotingo \
  --branch-pattern='^main$' \
  --build-config=cloudbuild.yaml \
  --project="$PROJECT_ID"

echo "==> 7/7: Create CD trigger (env repo, filtered to prod/**)"
gcloud builds triggers create github \
  --name=quotingo-cd \
  --repo-owner="$GITHUB_OWNER" \
  --repo-name=quotingo-env \
  --branch-pattern='^main$' \
  --included-files='prod/**' \
  --build-config=cloudbuild.yaml \
  --project="$PROJECT_ID"

echo ""
echo "Setup complete. Next steps:"
echo "  - Push a change to msafwankarim/quotingo to trigger the pipeline"
echo "  - Watch the CI build:  https://console.cloud.google.com/cloud-build/builds?project=$PROJECT_ID"
echo "  - After CD completes:  kubectl get svc -n quotingo"
