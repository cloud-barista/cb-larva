#!/bin/bash


# Update apt package list
echo 
echo =================================================
echo == update apt package list
echo =================================================
sleep 2
sudo apt update -y

# Upgrade apt package considering dependencies
echo 
echo =================================================
echo == upgrade apt package considering dependencies
echo =================================================
sleep 2
sudo apt dist-upgrade -y

# Install net-tools for ifconfig
echo =================================================
echo == Install net-tools
echo =================================================
sleep 2
sudo apt install net-tools -y

# Install bridge-utils
echo =================================================
echo == Install bridge-utils
echo =================================================
sleep 2
sudo apt install bridge-utils -y
