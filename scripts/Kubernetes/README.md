# Kubernetes cluster configuration scripts
The scripts here are used to configure a single global Kubernetes (K8s) cluster based on a cloud adaptive network.
The cloud adaptive network is a feature that cb-network strives to provide. 
This feature would help to communicate efficiently between VMs on multi-cloud.
We hope the cloud adaptive network also help to configure a K8s cluster globally.

There's an issue to configure a K8s cluster as follows:
- [Ready state of the cluster] Joining node, and then applying flannel 
- [NOT ready state of the cluster] Applying flannel, and then joining node

I'm not so sure the reason of the issue yet, but I'm trying to figure out the cause.