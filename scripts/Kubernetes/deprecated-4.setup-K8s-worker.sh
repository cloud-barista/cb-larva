#!/bin/bash

# Install k8s
echo
echo =================================================
echo == Install k8s
echo =================================================
echo "sudo apt install -y kubelet kubeadm kubectl kubernetes-cni"
sleep 1
sudo apt install -y kubelet kubeadm kubectl kubernetes-cni

# Join workers to the master
echo
echo =================================================
echo == Join workers to the master
echo =================================================
echo "Execute the result of 'kubeadm init' on the master"
echo "[An example of the result]"
echo "kubeadm join [YOUR_MASTER_IP_ADDRESS]:6443 --token xxxxxxxxxxxxxxxxxxxxxxx \ "
echo "    --discovery-token-ca-cert-hash sha256:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"