#!/bin/bash

# Install docker
echo
echo =================================================
echo == Install Docker engine
echo =================================================
# Install docker
echo "wget -qO- get.docker.com | sh"
sleep 1
wget -qO- get.docker.com | sh
# Check the installed docker
echo "sudo docker version"
sleep 1
sudo docker version

# Modify a user account for docker
echo
echo =================================================
echo == Modify a user account for docker
echo =================================================
echo "sudo usermod $USER -aG docker && newgrp docker"
sleep 1
sudo usermod $USER -aG docker && newgrp docker

# Change the default cgroups driver Docker uses from cgroups to systemd
# to allow systemd to act as the cgroups manager and
# ensure there is only one cgroup manager in use.
echo
echo =================================================
echo == Change the default cgoups drive Docker uses
echo =================================================
sleep 1
sudo cat > /etc/docker/daemon.json <<EOF
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
# Restart docker
echo "sudo systemctl restart docker"
sleep 1
sudo systemctl restart docker