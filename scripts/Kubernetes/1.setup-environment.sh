#!/bin/bash

# Allow iptables to see bridged traffic
echo
echo =================================================
echo == Allow iptables to see bridged traffic
echo =================================================
sleep 1
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
br_netfilter
EOF

cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF
sudo sysctl --system

# Add k8s repository
echo
echo =================================================
echo == Add kubernetes repostiory
echo =================================================
cat <<EOF
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add - && \
echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list
EOF
sleep 1
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add - && \
echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list

# Update apt package list
echo
echo =================================================
echo == Update apt package list
echo =================================================
echo "sudo apt update -y"
sleep 1
sudo apt update -y

# Upgrade apt package considering dependencies
echo
echo =================================================
echo == Upgrade apt package considering dependencies
echo =================================================
echo "apt dist-upgrade -y"
sleep 1
sudo apt dist-upgrade -y

echo "!!! Please reboot !!!"