#!/bin/bash

# Reset Kubernetes
echo
echo =================================================
echo == Reset Kubernetes
echo =================================================
echo "sudo kubeadm reset"
sleep 1
sudo kubeadm reset

echo "Some files could be removed manually."