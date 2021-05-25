#!/bin/bash

echo "Did you set the target repo and branch? IF NOT, quit within 5sec by ctrl+c"
sleep 5

ETCD_HOSTS=${1:-no}
CLADNET_ID=${2:-no}
HOST_ID=${3:-no}

if [ "${ETCD_HOSTS}" == "no" ] || [ "${CLADNET_ID}" == "no" ] || [ "${HOST_ID}" == "no" ]; then

  echo "Please, check parameters: etcd_hosts(${ETCD_HOSTS}), cladnet_id(${CLADNET_ID}), or host_id(${HOST_ID})"
  echo "The execution guide: ./build-agent.sh etcd_hosts(array) cladnet_id(string) host_id(string)"
  echo "An example: ./build-agent.sh '[\"xxx.xxx.xxx:xxxx\", \"xxx.xxx.xxx:xxxx\", \"xxx.xxx.xxx:xxxx\"]' xxx xxx"

else


# Prerequisites
echo "Step 1-1: Update apt"
# Update apt
sudo apt update -y


echo "Step 1-2: Install git"
# Install git
sudo apt install git -y


echo "Step 1-3: Install gcc"
#Install gcc
sudo apt install gcc -y


GOLANG_VERSION=1.16.4
echo "Step 1-4: Install and setup Golang ${GOLANG_VERSION}"
# Install golang by apt
# Install Go
wget https://dl.google.com/go/go${GOLANG_VERSION}.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz

# Set Go env (for next interactive shell)
echo "export PATH=${PATH}:/usr/local/go/bin" >> ${HOME}/.bashrc
echo "export GOPATH=${HOME}/go" >> ${HOME}/.bashrc
# Set Go env (for current shell)
export PATH=${PATH}:/usr/local/go/bin
export GOPATH=${HOME}/go

go version


# Download source code
echo "Step 2-1: Download cb-network source code"

cd ~

# master branch in upstream
# git clone master https://github.com/cloud-barista/cb-larva.git
# develop branch in upstream
# git clone -b develop https://github.com/cloud-barista/cb-larva.git
# (for development) A specific branch in forked repo
git clone -b develop https://github.com/cloud-barista/cb-larva.git


echo "Step 2-2: Build the cb-network agent"
# Change directory to where agent.go is located
cd ~/cb-larva/poc-cb-net/cmd/agent

# Build agent
# Note - Using the -ldflags parameter can help set variable values at compile time.
# Note - Using the -s and -w linker flags can strip the debugging information.
go build -mod=mod -a -ldflags '-s -w' -o agent


echo "Step 2-3: Copy the execution file of cb-network agent to $HOME/cb-network-agent"
# Create directory for execution
mkdir ~/cb-network-agent
# Copy the execution file of the cb-network agent
cp ~/cb-larva/poc-cb-net/cmd/agent/agent ~/cb-network-agent/


echo "Step 2-4: Generate config.yaml"
# Create directory for configuration files of the cb-network agent
mkdir ~/cb-network-agent/configs

# Change directory to configs
cd ~/cb-network-agent/configs

# Refine ${ETCD_HOSTS} because of parameter passing issue by json array string including ', ", and \.
REFINED_ETCD_HOSTS=${ETCD_HOSTS//\\/}
# Meaning: "//": replace every, "\\": backslash, "/": with, "": empty string

# Generate the config for the cb-network agent
cat <<EOF >./config.yaml
mqtt_broker:
  host: "xxx"
  port: "xxx"
  port_for_websocket: "xxx"

etcd_cluster:
  endpoints: ${REFINED_ETCD_HOSTS}

admin_web:
  host: "localhost"
  port: "9999"

cb_network:
  cladnet_id: "${CLADNET_ID}"
  host_id: "${HOST_ID}"
EOF


echo "Step 2-5: Generate log_conf.yaml"
# Generate the config for the cb-network agent
cat <<EOF >./log_conf.yaml
#### Config for CB-Log Lib. ####

cblog:
  ## true | false
  loopcheck: true # This temp method for development is busy wait. cf) cblogger.go:levelSetupLoop().

  ## debug | info | warn | error
  loglevel: trace # If loopcheck is true, You can set this online.

  ## true | false
  logfile: false

## Config for File Output ##
logfileinfo:
  filename: ./log/cblogs.log
  #  filename: $CBLOG_ROOT/log/cblogs.log
  maxsize: 10 # megabytes
  maxbackups: 50
  maxage: 31 # days
EOF


echo "Step 2-6: Clean up the source code of cb-network-agent"
rm -rf ~/cb-larva


echo "Step 3: Run cb-network agent"
cd ~/cb-network-agent
sudo ./agent

fi