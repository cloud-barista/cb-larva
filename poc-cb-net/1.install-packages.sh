#!/bin/bash

# Update apt pakage list
apt update -y

# Upgrade apt package considering dependencies
apt dist-upgrade -y

# Install packages
apt install -y build-essential fakeroot wget vim
