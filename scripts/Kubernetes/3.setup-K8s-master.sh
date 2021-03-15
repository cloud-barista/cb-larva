#!/bin/bash

IPprefix_by_netmask () {
   c=0 x=0$( printf '%o' ${1//./ } )
   while [ $x -gt 0 ]; do
       let c+=$((x%2)) 'x>>=1'
   done
   echo /$c ;
}

IPconfig_to_netaddr () {
	line=`ifconfig -a $1 | grep netmask | tr -s " "`
	ip=`echo $line | cut -f 2 -d " "`
	mask=`echo $line | cut -f 4 -d " "`

	IFS=. read -r io1 io2 io3 io4 <<< $ip
	IFS=. read -r mo1 mo2 mo3 mo4 <<< $mask
	NET_ADDR="$((io1 & mo1)).$(($io2 & mo2)).$((io3 & mo3)).$((io4 & mo4))"

	echo $NET_ADDR`IPprefix_by_netmask $mask` ;
}

if [ "$#" -ne 1 ]; then
	echo "Usage: $0 network_interface(e.g., eth0)"
	exit 2
fi

net_i=$1
found=`ifconfig -a $net_i 2> /dev/null`
if [ $? -eq 1 ]; then
	echo $0: $net_i interface not found
	exit 1
fi

MASTER_IP_ADDRESS=$(ifconfig $net_i | grep "inet " | awk '{print $2}')
POD_NETWORK_CIDR=$(IPconfig_to_netaddr $net_i)

# Install k8s
echo
echo =================================================
echo == Install k8s
echo =================================================
echo "sudo apt install -y kubelet kubeadm kubectl kubernetes-cni"
sleep 1
sudo apt install -y kubelet kubeadm kubectl kubernetes-cni

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
echo "kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml"
sleep 1
kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml