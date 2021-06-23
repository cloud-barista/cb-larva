#!/bin/bash

# Prerequisites
echo "Step 1: Update apt"
# Update apt
sudo apt update -y

echo "Step 2: Install git"
# Install git
sudo apt install git -y

GOLANG_VERSION=1.16.4
echo "Step 3: Install and setup Golang ${GOLANG_VERSION}"
# Install golang by apt
# Install Go
if [ ! -d /usr/local/go ]; then
  wget https://dl.google.com/go/go${GOLANG_VERSION}.linux-amd64.tar.gz
  sudo tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz
  # Set Go env (for next interactive shell)
  echo "export PATH=\${PATH}:/usr/local/go/bin" >> ${HOME}/.bashrc
  echo "export GOPATH=\${HOME}/go" >> ${HOME}/.bashrc
fi

# Set Go env (for current shell)
export PATH=${PATH}:/usr/local/go/bin
export GOPATH=${HOME}/go

echo "HOME: ${HOME}"
whoami
go version