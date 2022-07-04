#!/bin/bash

# IPprefix_by_netmask () {
#    c=0 x=0$( printf '%o' ${1//./ } )
#    while [ $x -gt 0 ]; do
#        let c+=$((x%2)) 'x>>=1'
#    done
#    echo /$c ;
# }

# IPconfig_to_netaddr () {
# 	line=`ifconfig -a $1 | grep netmask | tr -s " "`
# 	ip=`echo $line | cut -f 2 -d " "`
# 	mask=`echo $line | cut -f 4 -d " "`

# 	IFS=. read -r io1 io2 io3 io4 <<< $ip
# 	IFS=. read -r mo1 mo2 mo3 mo4 <<< $mask
# 	NET_ADDR="$((io1 & mo1)).$(($io2 & mo2)).$((io3 & mo3)).$((io4 & mo4))"

# 	echo $NET_ADDR`IPprefix_by_netmask $mask` ;
# }

# if [ "$#" -ne 1 ]; then
# 	echo "Usage: $0 network_interface(e.g., eth0)"
# 	exit 2
# fi

# net_i=$1
# found=`ifconfig -a $net_i 2> /dev/null`
# if [ $? -eq 1 ]; then
# 	echo $0: $net_i interface not found
# 	exit 1
# fi

# MASTER_IP_ADDRESS=$(ifconfig $net_i | grep "inet " | awk '{print $2}')
# POD_NETWORK_CIDR=$(IPconfig_to_netaddr $net_i)


# Assign default values for network parameters
DEFAULT_NETWORK_INTERFACE="cbnet0"
DEFAULT_POD_NETWORK_CIDR="10.77.0.0/16"

NETWORK_INTERFACE=$DEFAULT_NETWORK_INTERFACE
POD_NETWORK_CIDR=$DEFAULT_POD_NETWORK_CIDR

# Update values for network parameters by named input parameters (i, c)
while getopts ":i:c:" opt; do
  case $opt in
    i) NETWORK_INTERFACE="$OPTARG"
    ;;
    c) POD_NETWORK_CIDR="$OPTARG"
    ;;
    \?) echo "Invalid option -$OPTARG (Use: -i for NETWORK_INTERFACE, -c for POD_NETWORK_CIDR)" >&2
    ;;
  esac
done

MASTER_IP_ADDRESS=$(ifconfig ${NETWORK_INTERFACE} | grep "inet " | awk '{print $2}')

printf "[Network env variables for this Kubernetes cluster installation]\nNETWORK_INTERFACE=%s\nMASTER_IP_ADDRESS=%s\nPOD_NETWORK_CIDR=%s\n\n" "$NETWORK_INTERFACE" "$MASTER_IP_ADDRESS" "$POD_NETWORK_CIDR"

if [ -z "$MASTER_IP_ADDRESS" ]
then
      echo "Warning! can not find NETWORK_INTERFACE $NETWORK_INTERFACE from ifconfig."
      echo ""
      echo "You need to provide an appropriate network interface."
      echo "Please check ifconfig and find an interface (Ex: ens3, ens4, eth0, ...)"
      echo "Then, provide the interface to this script with parameter option '-i' (ex: ./${0##*/} -i ens3)"
      echo ""
      echo "See you again.. :)"
      exit 0
fi

# Do this due to temporal issue/bug (https://github.com/containerd/containerd/issues/4581)
sudo rm /etc/containerd/config.toml
sudo systemctl restart containerd

# Initialize k8s cluster on a Master
echo
echo =================================================
echo == Initialize k8s cluster on a Master
echo =================================================
echo "sudo kubeadm init --apiserver-advertise-address $MASTER_IP_ADDRESS --pod-network-cidr=$POD_NETWORK_CIDR"
sleep 1
sudo kubeadm init --apiserver-advertise-address $MASTER_IP_ADDRESS --pod-network-cidr=$POD_NETWORK_CIDR

# Allow a regular user to control the cluster on a master
echo
echo =================================================
echo == Allow a regular user to control the cluster on a master
echo =================================================
echo "mkdir -p $HOME/.kube"
sleep 1
mkdir -p $HOME/.kube

echo "sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config"
sleep 1
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config

echo "sudo chown $(id -u):$(id -g) $HOME/.kube/config"
sleep 1
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# Install container network plugins on a master
echo
echo =================================================
echo == Install conatiner network plugins on a master
echo =================================================
#echo "kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml"
#sleep 1
#kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
echo "kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml"
sleep 1
kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml

echo "Go to each Kubernetes node and join the nodes to K8s master accoring to the following command"
echo ""

sudo kubeadm token create --print-join-command
