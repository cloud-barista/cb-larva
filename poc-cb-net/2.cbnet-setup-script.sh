#!/bin/bash


# Global variables
PHYSICAL_ETHERNET_DEVICE="eth0"
VIRTUAL_ETHERNET_DEVICE_0="veth0"
VIRTUAL_ETHERNET_DEVICE_1="veth1"
VIRTUAL_ETHERNET_IP="10.0.0.25/24"
BRIDGE_NAME="cbnet0"
BRIDGE_IP="192.168.77.1/24"
DEFAULT_GATEWAY="192.168.0.1"


# Add a virtual ethernet interface !!! To be updated
# Virtual Ethernet interfaces are always come in pairs, and they are connected like a tube.
# ip link add veth0 type veth peer name veth1
ip link add ${VIRTUAL_ETHERNET_DEVICE_0} type veth peer name ${VIRTUAL_ETHERNET_DEVICE_1}

# Set up the viratual ethernet interface pair
# Command: ip link set [INTERFACE] up
# e.g., ip link set veth0 up
ip link set ${VIRTUAL_ETHERNET_DEVICE_0} up
ip link set ${VIRTUAL_ETHERNET_DEVICE_1} up

# Add bridge
# Command: brctl addbr [BRIDGE]
brctl addbr ${BRIDGE_NAME}

# Set up the bridge interface
ip link set up ${BRIDGE_NAME}

# Assign IP address to bridge
# e.g., ip addr add 192.168.0.25/24 dev cbnet0
ip addr add ${BRIDGE_IP} dev ${BRIDGE_NAME}

# Set up default gateway
# e.g., route add default gw 192.168.0.1 dev cbnet0
route add default gw ${DEFAULT_GATEWAY} dev ${BRIDGE_NAME}

# Set up Domain Name System (DNS)
# It's temporary.
echo "nameserver 8.8.8.8" > /etc/resolve.conf

# Add interface to bridge (Add eth0 and veth0 to cbnet0)
# Command: brctl addif [BRIDGE] [DEVICE]
# e.g., brctl addif cbnet0 veth0
brctl addif ${BRIDGE_NAME} ${PHYSICAL_ETHERNET_DEVICE}
brctl addif ${BRIDGE_NAME} ${VIRTUAL_ETHERNET_DEVICE_0}


# Assign IP address to virtal eth
# e.g., ip addr add 10.0.0.25/24 dev veth0
ip addr add ${VIRTUAL_ETHERNET_IP} dev ${VIRTUAL_ETHERNET_DEVICE_1}