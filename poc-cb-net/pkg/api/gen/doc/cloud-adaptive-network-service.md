# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [cbnetwork/cloud_adaptive_network.proto](#cbnetwork/cloud_adaptive_network.proto)
    - [AvailableIPv4PrivateAddressSpaces](#cbnet.AvailableIPv4PrivateAddressSpaces)
    - [CLADNetID](#cbnet.CLADNetID)
    - [CLADNetSpecification](#cbnet.CLADNetSpecification)
    - [CLADNetSpecifications](#cbnet.CLADNetSpecifications)
    - [DeletionResult](#cbnet.DeletionResult)
    - [IPNetworks](#cbnet.IPNetworks)
  
    - [CloudAdaptiveNetworkService](#cbnet.CloudAdaptiveNetworkService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="cbnetwork/cloud_adaptive_network.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cbnetwork/cloud_adaptive_network.proto

Messages and services of Cloud Adaptive Network (shortly CLADNet) are defined in this proto.
 
The messages are described at first.
The service is described next.

NOTE - The auto-generated API document describes this proto in alphabetical order.


<a name="cbnet.AvailableIPv4PrivateAddressSpaces"></a>

### AvailableIPv4PrivateAddressSpaces

It represents available IPv4 private address spaces
(also known as CIDR block, CIDR range, IP address range).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| recommended_ipv4_private_address_space | [string](#string) |  |  |
| address_space10s | [string](#string) | repeated |  |
| address_space172s | [string](#string) | repeated |  |
| address_space192s | [string](#string) | repeated |  |






<a name="cbnet.CLADNetID"></a>

### CLADNetID

It represents An ID of Cloud Adaptive Network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |






<a name="cbnet.CLADNetSpecification"></a>

### CLADNetSpecification

It represents a specification of Cloud Adaptive Network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| ipv4_address_space | [string](#string) |  |  |
| description | [string](#string) |  |  |






<a name="cbnet.CLADNetSpecifications"></a>

### CLADNetSpecifications

It represents a list of Cloud Adaptive Network specifications.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cladnet_specifications | [CLADNetSpecification](#cbnet.CLADNetSpecification) | repeated |  |






<a name="cbnet.DeletionResult"></a>

### DeletionResult

It represents a result of attempt to delete a Cloud Adaptive Network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| is_succeeded | [bool](#bool) |  |  |
| message | [string](#string) |  |  |
| cladnet_specification | [CLADNetSpecification](#cbnet.CLADNetSpecification) |  |  |






<a name="cbnet.IPNetworks"></a>

### IPNetworks

It represents A list of IP networks (e.g., 10.0.0.0/8).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip_networks | [string](#string) | repeated |  |





 

 

 


<a name="cbnet.CloudAdaptiveNetworkService"></a>

### CloudAdaptiveNetworkService

Service for handling Cloud Adaptive Network

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| sayHello | [.google.protobuf.Empty](#google.protobuf.Empty) | [.google.protobuf.StringValue](#google.protobuf.StringValue) | Used to say hello (for testing). Pass in nothing and return a say-hello message. |
| getCLADNet | [CLADNetID](#cbnet.CLADNetID) | [CLADNetSpecification](#cbnet.CLADNetSpecification) | Used to get a Cloud Adaptive Network specification. Pass in an ID of Cloud Adaptive Networkand return a Cloud Adaptive Network specification. |
| getCLADNetList | [.google.protobuf.Empty](#google.protobuf.Empty) | [CLADNetSpecifications](#cbnet.CLADNetSpecifications) | Used to get a list of Cloud Adaptive Network specifications. Pass in nothing and return a list of Cloud Adaptive Network specifications. |
| createCLADNet | [CLADNetSpecification](#cbnet.CLADNetSpecification) | [CLADNetSpecification](#cbnet.CLADNetSpecification) | Used to create a new Cloud Adaptive Network. Pass in a specification of Cloud Adaptive Network and return the specification of Cloud Adaptive Network. |
| recommendAvailableIPv4PrivateAddressSpaces | [IPNetworks](#cbnet.IPNetworks) | [AvailableIPv4PrivateAddressSpaces](#cbnet.AvailableIPv4PrivateAddressSpaces) | Used to recommend available IPv4 private address spaces for Cloud Adaptive Network. Pass in a list of IP networks (e.g., [&#34;10.10.10.10/14&#34;, &#34;192.168.20.20/26&#34;, ....]) and return available IPv4 private address spaces |
| deleteCLADNet | [CLADNetID](#cbnet.CLADNetID) | [DeletionResult](#cbnet.DeletionResult) | [To be provided] Used to delete a Cloud Adaptive Network Pass in an ID of Cloud Adaptive Network and return a result of attempt to delete a Cloud Adaptive Network. |
| updateCLADNet | [CLADNetSpecification](#cbnet.CLADNetSpecification) | [CLADNetSpecification](#cbnet.CLADNetSpecification) | [To be provided] Used to update a Cloud Adaptive Network Pass in a specification of Cloud Adaptive Network and return the specification of Cloud Adaptive Network. |

 



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

