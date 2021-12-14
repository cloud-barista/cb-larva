#!/bin/bash

###########################################################
# Example: Example: Deploying Weave Scope                 #
###########################################################

echo ""
echo =================================================
echo "== Start up Weave Scope"
echo =================================================

# Step 1: Deploy Weave Scope
echo ""
echo "== Step 1: Deploy Weave Scope"
echo "kubectl apply -f 'https://cloud.weave.works/launch/k8s/weavescope.yaml?k8s-service-type=NodePort'"
sleep 1
kubectl apply -f 'https://cloud.weave.works/launch/k8s/weavescope.yaml?k8s-service-type=NodePort'

echo ""
echo =================================================
echo "== Check if Weave Scope is deployed"
echo =================================================

# Step 2: Check NodePort of Weave Scope
echo ""
echo "== Step 2: Check NodePort of Weave Scope"
echo "kubectl get service -n weave"
sleep 1
kubectl get service -n weave
