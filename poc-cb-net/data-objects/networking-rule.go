package dataobjects

import "fmt"

type NetworkingRule struct {
	ID       []string
	CBNet    []string
	CBNetIP  []string
	PublicIP []string
}

func (netrule *NetworkingRule) AppendRule(ID string, CBNet string, CBNetIP string, PublicIP string) {

	fmt.Println(ID, CBNet, CBNetIP, PublicIP)
	if netrule.contains(netrule.ID, ID) { // If ID exists, update rule
		index := netrule.GetIndexOfID(ID)
		netrule.CBNet[index] = CBNet
		netrule.CBNetIP[index] = CBNetIP
		netrule.PublicIP[index] = PublicIP
	} else { // Else append rule
		netrule.ID = append(netrule.ID, ID)
		netrule.CBNet = append(netrule.CBNet, CBNet)
		netrule.CBNetIP = append(netrule.CBNetIP, CBNetIP)
		netrule.PublicIP = append(netrule.PublicIP, PublicIP)
	}
}

func (netrule NetworkingRule) GetIndexOfID(ID string) int {
	return netrule.find(netrule.ID, ID)
}

func (netrule NetworkingRule) GetIndexOfCBNet(CBNet string) int {
	return netrule.find(netrule.CBNet, CBNet)
}

func (netrule NetworkingRule) GetIndexOfCBNetIP(CBNetIP string) int {
	return netrule.find(netrule.CBNetIP, CBNetIP)
}

func (netrule NetworkingRule) GetIndexOfPublicIP(PublicIP string) int {
	return netrule.find(netrule.PublicIP, PublicIP)
}

func (netrule NetworkingRule) find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

func (netrule NetworkingRule) contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
