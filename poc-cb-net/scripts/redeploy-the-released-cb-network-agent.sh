#!/bin/bash

ETCD_HOSTS=${1:-no}
CLADNET_ID=${2:-no}
HOST_NAME=${3:-no}

if [ "${ETCD_HOSTS}" == "no" ] || [ "${CLADNET_ID}" == "no" ]; then
  echo "Please, check parameters: etcd_hosts(${ETCD_HOSTS}) or cladnet_id(${CLADNET_ID})"
  echo "The execution guide: ./get-and-run-agent.sh etcd_hosts(Required) cladnet_id(Required) host_name(Optional)"
  echo "An example: ./get-and-run-agent.sh '[\"xxx.xxx.xxx:xxxx\", \"xxx.xxx.xxx:xxxx\", \"xxx.xxx.xxx:xxxx\"]' xxx xxx"

else

echo "Step 1: Check status of the cb-network agent service"
sudo systemctl status cb-network-agent.service
sleep 1

echo "Step 2: Stop the cb-network agent service"
sudo systemctl stop cb-network-agent.service
sleep 1

echo "Step 3: Disable the cb-network agent service"
sudo systemctl disable cb-network-agent.service
sleep 1

if [ "${HOST_NAME}" == "no" ]; then
  echo "No input host_name(${HOST_NAME}). The hostname of node is used."
  HOST_NAME=""
fi

echo "Step 4: Get the execution file of cb-network agent to $HOME/cb-network-agent"
# Create directory for execution
mkdir ~/cb-network-agent

# Change directory
cd ~/cb-network-agent


# Get the execution file of the cb-network agent
wget -q --no-cache https://github.com/cloud-barista/cb-larva/releases/download/v0.0.14/agent-v0.0.14 -O agent
ls -al agent

# Change mode
chmod 755 agent


echo "Step 5: Generate config.yaml"
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

# A config for the cb-network service and cb-network admin-web as follows:
service:
  endpoint: "localhost:8053"
  port: "8053"

# A config for the cb-network admin-web as follows:
admin_web:
  host: "localhost"
  port: "8054"

# A config for the cb-network agent as follows:
cb_network:
  cladnet_id: "${CLADNET_ID}"
  host: # for each host
    name: "${HOST_NAME}" # if name is "" (empty string), the cb-network agent will use hostname.
    network_interface_name: "" # if network_interface_name is "" (empty string), the cb-network agent will use "cbnet0".
    tunneling_port: "" # if network_interface_port is "" (empty string), the cb-network agent will use "8055".
    is_encrypted: false  # false is default.

# A config for the demo-client as follows:
service_call_method: "grpc" # i.e., "rest" / "grpc"

EOF


echo "Step 6: Generate log_conf.yaml"
# Generate the config for the cb-network agent
cat <<EOF >./log_conf.yaml
#### Config for CB-Log Lib. ####

cblog:
  ## true | false
  loopcheck: true # This temp method for development is busy wait. cf) cblogger.go:levelSetupLoop().

  ## trace | debug | info | warn | error
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

echo "Step 7: Create a script to run and stop a cb-network agent"

cd ~/cb-network-agent

# The script to run
cat <<EOF >./run-cb-network-agent.sh
#!/bin/bash

sudo ${HOME}/cb-network-agent/agent

EOF

sudo chmod 755 run-cb-network-agent.sh

# The script to stop
cat <<EOF >./stop-cb-network-agent.sh
#!/bin/bash

sudo pkill -9 -f cb-network-agent

EOF

sudo chmod 755 stop-cb-network-agent.sh


echo "Step 8: Create a service file of the cb-network agent"

## Detect OS ID (ubuntu / centos) without double quote sign
OS_ID=$(awk -F= '$1=="ID" { print $2 ;}' /etc/os-release | tr -d \")

SYSTEMD_PATH=""

case "$OS_ID" in
  ubuntu*) 
  echo "ubuntu"
  SYSTEMD_PATH="/lib/systemd/system/cb-network-agent.service"
  ;;

  centos*)  
  echo "centos" 
  SYSTEMD_PATH="/usr/lib/systemd/system/cb-network-agent.service"
  ;;

  *)
  echo "unknown: $OS_ID" 
  ;;
esac

# if systemd path is not ""
if [ "${OS_ID}" == "ubuntu" ] || [ "${OS_ID}" == "centos" ]; then
cat <<EOF | sudo tee -a ${SYSTEMD_PATH}

[Unit]
Description=Service of cb-network agent

[Service]
Type=simple
ExecStart=${HOME}/cb-network-agent/run-cb-network-agent.sh
ExecStop=${HOME}/cb-network-agent/stop-cb-network-agent.sh
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

echo "Step 9: Start the cb-network agent service"
sudo systemctl start cb-network-agent.service
sleep 1

echo "Step 10: enable start on boot of the cb-network agent service"
sudo systemctl enable cb-network-agent.service
sleep 1
fi

fi
