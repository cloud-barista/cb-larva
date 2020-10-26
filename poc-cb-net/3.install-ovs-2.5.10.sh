#!/bin/bash

# Get Open vSwitch 2.5.10
wget https://www.openvswitch.org/releases/openvswitch-2.5.10.tar.gz

# Decompress
tar -zxvf openvswitch-2.5.10.tar.gz

# Enter to the directory
cd openvswitch-2.5.10

# copy dhparams.c for backup
cp ./lib/dhparams.c ./lib/dhparmas.c.backup
# Delete static for debug (in 3 function declaration in .h file, there are no static keywords)
sed -i 's/static //g' ./lib/dhparams.c

# Build Open vSwitch 
DEB_BUILD_OPTIONS='parallel=8 nocheck' fakeroot debian/rules binary
