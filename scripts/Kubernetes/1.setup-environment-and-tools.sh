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

## Upgrade apt package considering dependencies
#echo
#echo =================================================
#echo == Upgrade apt package considering dependencies
#echo =================================================
#echo "sudo DEBIAN_FRONTEND=noninteractive apt dist-upgrade -y"
#sleep 1
#sudo DEBIAN_FRONTEND=noninteractive apt dist-upgrade -y

# Install apt-utils
echo =================================================
echo == Install apt-utils
echo =================================================
echo "sudo DEBIAN_FRONTEND=noninteractive apt install -y apt-utils"
sleep 1
sudo DEBIAN_FRONTEND=noninteractive apt install -y apt-utils

# Install Kubernetes packages
echo
echo =================================================
echo == Install Kubernetes packages
echo =================================================
echo "sudo DEBIAN_FRONTEND=noninteractive apt install -y kubelet kubeadm kubectl kubernetes-cni"
sleep 1
sudo DEBIAN_FRONTEND=noninteractive apt install -y kubelet kubeadm kubectl kubernetes-cni

# Install docker
echo
echo =================================================
echo == Install Docker engine
echo =================================================
echo "wget -qO- get.docker.com | sh"
sleep 1
wget -qO- get.docker.com | sh
# Check the installed docker
echo "sudo docker version"
sleep 1
sudo docker version

# Setup to manage Docker as a non-root user
echo
echo =================================================
echo == Setup to manage Docker as a non-root user
echo == ** The manual LOGOUT and RE-LOGIN are REQUIRED
echo =================================================
echo "sudo groupadd docker"
sleep 1
sudo groupadd docker
echo "sudo usermod $USER -aG docker"
sleep 1
sudo usermod $USER -aG docker

# Change the default cgroups driver Docker uses from cgroups to systemd
# to allow systemd to act as the cgroups manager and
# ensure there is only one cgroup manager in use.
echo
echo =================================================
echo == Change the default cgoups drive Docker uses
echo =================================================
sleep 1
sudo cat <<EOF | sudo tee /etc/docker/daemon.json
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m"
  },
  "storage-driver": "overlay2"
}
EOF

# Make a directory
echo "sudo mkdir -p /etc/systemd/system/docker.service.d"
sleep 1
sudo mkdir -p /etc/systemd/system/docker.service.d

# Reload daemon
echo "sudo systemctl daemon-reload"
sleep 1
sudo systemctl daemon-reload

# Configure Docker to start on boot
echo "Configure Docker to start on boot"
sleep 1
sudo systemctl enable docker.service

# Restart docker
echo "sudo systemctl restart docker"
sleep 1
sudo systemctl restart docker

#####
# Do this due to temporal issue/bug (https://github.com/containerd/containerd/issues/4581)
sudo rm /etc/containerd/config.toml
sudo systemctl restart containerd

echo "!!! Please reboot !!!"
echo "!!! Please reboot !!!"
echo "!!! Please reboot !!!"
