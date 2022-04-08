# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [cbnetwork/cloud_adaptive_network.proto](#cbnetwork/cloud_adaptive_network.proto)
    - [AvailableIPv4PrivateAddressSpaces](#cbnet.v1.AvailableIPv4PrivateAddressSpaces)
    - [CLADNetID](#cbnet.v1.CLADNetID)
    - [CLADNetSpecification](#cbnet.v1.CLADNetSpecification)
    - [CLADNetSpecifications](#cbnet.v1.CLADNetSpecifications)
    - [DeletionResult](#cbnet.v1.DeletionResult)
    - [IPNetworks](#cbnet.v1.IPNetworks)
  
    - [CloudAdaptiveNetworkService](#cbnet.v1.CloudAdaptiveNetworkService)
    - [SystemManagementService](#cbnet.v1.SystemManagementService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="cbnetwork/cloud_adaptive_network.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cbnetwork/cloud_adaptive_network.proto
Messages and services of Cloud Adaptive Network (shortly CLADNet) are defined in this proto.
 
The messages are described at first.
The service is described next.

NOTE - The auto-generated API document describes this proto in alphabetical order.


<a name="cbnet.v1.AvailableIPv4PrivateAddressSpaces"></a>

### AvailableIPv4PrivateAddressSpaces
It represents available IPv4 private address spaces
(also known as CIDR block, CIDR range, IP address range).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| recommended_ipv4_private_address_space | [string](#string) |  | A recommended IPv4 address space |
| address_space10s | [string](#string) | repeated | All available Ipv4 address space in 10.0.0.0/8 |
| address_space172s | [string](#string) | repeated | All available Ipv4 address space in 172.16.0.0/12 |
| address_space192s | [string](#string) | repeated | All available Ipv4 address space in 192.168.0.0/16 |






<a name="cbnet.v1.CLADNetID"></a>

### CLADNetID
It represents an ID of Cloud Adaptive Network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |






<a name="cbnet.v1.CLADNetSpecification"></a>

### CLADNetSpecification
It represents a specification of Cloud Adaptive Network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | ID of Cloud Adaptive Network |
| name | [string](#string) |  | name of Cloud Adaptive Network |
| ipv4_address_space | [string](#string) |  | IPv4 address space (e.g., 192.168.0.0/24) of Cloud Adaptive Network |
| description | [string](#string) |  | Description of Cloud Adaptive Network |






<a name="cbnet.v1.CLADNetSpecifications"></a>

### CLADNetSpecifications
It represents a list of Cloud Adaptive Network specifications.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cladnet_specifications | [CLADNetSpecification](#cbnet.v1.CLADNetSpecification) | repeated | A list of Cloud Adaptive Network specification |






<a name="cbnet.v1.DeletionResult"></a>

### DeletionResult
It represents a result of attempt to delete a Cloud Adaptive Network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| is_succeeded | [bool](#bool) |  | Success or failure |
| message | [string](#string) |  | Message |
| cladnet_specification | [CLADNetSpecification](#cbnet.v1.CLADNetSpecification) |  | A specification of the target Cloud Adaptive Network |






<a name="cbnet.v1.IPNetworks"></a>

### IPNetworks
It represents a list of IP networks (e.g., 10.10.1.5/16).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip_networks | [string](#string) | repeated |  |





 

 

 


<a name="cbnet.v1.CloudAdaptiveNetworkService"></a>

### CloudAdaptiveNetworkService
Service for handling Cloud Adaptive Network

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| getCLADNet | [CLADNetID](#cbnet.v1.CLADNetID) | [CLADNetSpecification](#cbnet.v1.CLADNetSpecification) | Get a Cloud Adaptive Network specification |
| getCLADNetList | [.google.protobuf.Empty](#google.protobuf.Empty) | [CLADNetSpecifications](#cbnet.v1.CLADNetSpecifications) | Get a list of Cloud Adaptive Network specifications |
| createCLADNet | [CLADNetSpecification](#cbnet.v1.CLADNetSpecification) | [CLADNetSpecification](#cbnet.v1.CLADNetSpecification) | Create a new Cloud Adaptive Network |
| deleteCLADNet | [CLADNetID](#cbnet.v1.CLADNetID) | [DeletionResult](#cbnet.v1.DeletionResult) | [To be provided] Delete a Cloud Adaptive Network |
| updateCLADNet | [CLADNetSpecification](#cbnet.v1.CLADNetSpecification) | [CLADNetSpecification](#cbnet.v1.CLADNetSpecification) | [To be provided] Update a Cloud Adaptive Network |
| recommendAvailableIPv4PrivateAddressSpaces | [IPNetworks](#cbnet.v1.IPNetworks) | [AvailableIPv4PrivateAddressSpaces](#cbnet.v1.AvailableIPv4PrivateAddressSpaces) | Recommend available IPv4 private address spaces for Cloud Adaptive Network |


<a name="cbnet.v1.SystemManagementService"></a>

### SystemManagementService
Service for handling System Management

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| health | [.google.protobuf.Empty](#google.protobuf.Empty) | [.google.protobuf.StringValue](#google.protobuf.StringValue) | Checks service health |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

