#!/bin/bash

# Get pods
echo
echo =================================================
echo == Get pods
echo =================================================
#sleep 2
echo "kubectl get pods --namespace kube-system"
kubectl get pods --namespace kube-system

# Get nodes
echo
echo =================================================
echo == Get nodes
echo =================================================
#sleep 2
echo "kubectl get nodes"
kubectl get nodes