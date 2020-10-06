#!/bin/bash


# Global variables
physical_ethernet_device = "eth0"
virtual_ethernet_device_0 = "veth0"
virtual_ethernet_device_1 = "veth1"
virtual_ethernet_IP = "10.0.0.25/24"
bridge_name = "cbnet0"
bridge_IP = "192.168.77.1/24"
default_gateway = "192.168.0.1"


# Add a virtual ethernet interface !!! To be updated
# Virtual Ethernet interfaces are always come in pairs, and they are connected like a tube.
# ip link add veth0 type veth peer name veth1
ip link add ${virtual_ethernet_device_0} type veth peer name ${virtual_ethernet_device_1}

# Set up the viratual ethernet interface pair
# Command: ip link set [INTERFACE] up
# e.g., ip link set veth0 up
ip link set ${virtual_ethernet_device_0} up
ip link set ${virtual_ethernet_device_1} up

# Add bridge
# Command: brctl addbr [BRIDGE]
brctl addbr ${bridge_name}

# Set up the bridge interface
ip link set up ${bridge_name}

# Assign IP address to bridge
# e.g., ip addr add 192.168.0.25/24 dev cbnet0
ip addr add ${bridge_IP} dev ${bridge_name}

# Set up default gateway
# e.g., route add default gw 192.168.0.1 dev cbnet0
route add default gw ${default_gateway} dev ${bridge_name}

# Set up Domain Name System (DNS)
# It's temporary.
echo "nameserver 8.8.8.8" > /etc/resolve.conf

# Add interface to bridge (Add eth0 and veth0 to cbnet0)
# Command: brctl addif [BRIDGE] [DEVICE]
# e.g., brctl addif cbnet0 veth0
brctl addif ${bridge_name} ${physical_ethernet_device}
brctl addif ${bridge_name} ${virtual_ethernet_device_0}


# Assign IP address to virtal eth
# e.g., ip addr add 10.0.0.25/24 dev veth0
ip addr add ${virtual_ethernet_IP} dev ${virtual_ethernet_device_1}