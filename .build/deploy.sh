#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

NAMESPACE=netlib

if [[ "${BRANCH_NAME:-""}" != "main" ]]; then
  exit 0
fi

if [[ "${PROJECT_ID:-""}" == "" ]]; then
  echo "PROJECT_ID is not set"
  exit 1
fi

if [[ "${CLOUDSDK_COMPUTE_ZONE:-""}" == "" ]]; then
  echo "CLOUDSDK_COMPUTE_ZONE is not set"
  exit 1
fi

if [[ "${CLUSTER:-""}" == "" ]]; then
  echo "CLUSTER is not set"
  exit 1
fi

if [[ "${COMMIT_SHA:-""}" == "" ]]; then
  COMMIT_SHA=$(git rev-parse HEAD)
  export COMMIT_SHA
fi

gcloud container clusters get-credentials --project="$PROJECT_ID" --zone="$CLOUDSDK_COMPUTE_ZONE" "$CLUSTER"

kubectl version

echo "Applying secrets..."
sops --decrypt "manifest/secrets.yaml" | kubectl apply -n "$NAMESPACE" --validate -f -

echo "Deploying..."
kubectl kustomize "manifest" | envsubst | kubectl apply -n "$NAMESPACE" --validate -f -

echo "Done"
