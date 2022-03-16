#!/bin/bash

echo "Did you set the target repo and branch? IF NOT, quit within 5sec by ctrl+c"
sleep 5

ETCD_HOSTS=${1:-no}
CLADNET_ID=${2:-no}
HOST_ID=${3:-no}

if [ "${ETCD_HOSTS}" == "no" ] || [ "${CLADNET_ID}" == "no" ]; then
  echo "Please, check parameters: etcd_hosts(${ETCD_HOSTS}) or cladnet_id(${CLADNET_ID})"
  echo "The execution guide: ./get-and-run-agent.sh etcd_hosts(Required) cladnet_id(Required) host_id(Optional)"
  echo "An example: ./get-and-run-agent.sh '[\"xxx.xxx.xxx:xxxx\", \"xxx.xxx.xxx:xxxx\", \"xxx.xxx.xxx:xxxx\"]' xxx xxx"

else


if [ "${HOST_ID}" == "no" ]; then
  echo "No input host_id(${HOST_ID}). The hostname of node is used."
  HOST_ID=""
fi

echo "Step 1: Get the execution file of cb-network agent to $HOME/cb-network-agent"
# Create directory for execution
mkdir ~/cb-network-agent

# Change directory
cd ~/cb-network-agent


# Get the execution file of the cb-network agent
wget -q http://alvin-mini.iptime.org:18000/agent
ls -al agent

# Change mode
chmod 755 agent


echo "Step 2: Generate config.yaml"
# Create directory for configuration files of the cb-network agent
mkdir ~/cb-network-agent/config

# Change directory to config
cd ~/cb-network-agent/config

# Refine ${ETCD_HOSTS} because of parameter passing issue by json array string including ', ", and \.
REFINED_ETCD_HOSTS=${ETCD_HOSTS//\\/}
# Meaning: "//": replace every, "\\": backslash, "/": with, "": empty string

# Generate the config for the cb-network agent
cat <<EOF >./config.yaml
# A config for the both cb-network controller and agent as follows:
etcd_cluster:
  endpoints: ${REFINED_ETCD_HOSTS}

# A config for the cb-network AdminWeb as follows:
admin_web:
  host: "localhost"
  port: "9999"

# A config for the cb-network agent as follows:
cb_network:
  cladnet_id: "${CLADNET_ID}"
  host_id: "${HOST_ID}"

# A config for the grpc as follows:
grpc:
  service_endpoint: "localhost:8089"
  server_port: "8089"
  gateway_port: "8088"

EOF


echo "Step 3: Generate log_conf.yaml"
# Generate the config for the cb-network agent
cat <<EOF >./log_conf.yaml
#### Config for CB-Log Lib. ####

cblog:
  ## true | false
  loopcheck: true # This temp method for development is busy wait. cf) cblogger.go:levelSetupLoop().

  ## trace | debug | info | warn | error
  loglevel: debug # If loopcheck is true, You can set this online.

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


echo "Step 4: Configure cb-network agent to start on boot"
# Set it to the .bashrc
sudo cat <<EOF | sudo tee -a ${HOME}/.bashrc

IS_RUNNING=\$(sudo ps -aux | grep "sudo ./agent" | grep -v grep)

if [ "\$?" == 1 ]; then
  cd \${HOME}/cb-network-agent
  nohup sudo ./agent > /dev/null 2>&1 &
  cd \${HOME}
fi
EOF


echo "Step 5: Terminate the cb-network agent if it is running"
sudo pkill -9 -ef ./agent


echo "Step 6: Run cb-network agent in background"
cd ~/cb-network-agent
# nohup : HUP(hangup), doesn't terminate a process run by the command after stty hangs
# /dev/null : redirect stdout (Standard ouput) to /dev/null i.e discard/silent the output by command
# 2>&1 : specify 2>&1 to redirect stderr to the same place (&1 means /dev/null)
# (The last)& : run the command as background process
nohup sudo ./agent > /dev/null 2>&1 &


fi
