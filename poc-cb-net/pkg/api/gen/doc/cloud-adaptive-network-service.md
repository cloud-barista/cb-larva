# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [cbnetwork/cloud_adaptive_network.proto](#cbnetwork/cloud_adaptive_network.proto)
    - [AvailableIPv4PrivateAddressSpaces](#cbnet.AvailableIPv4PrivateAddressSpaces)
    - [CLADNetID](#cbnet.CLADNetID)
    - [CLADNetSpecification](#cbnet.CLADNetSpecification)
    - [CLADNetSpecifications](#cbnet.CLADNetSpecifications)
    - [IPNetworks](#cbnet.IPNetworks)
  
    - [CloudAdaptiveNetwork](#cbnet.CloudAdaptiveNetwork)
  
- [Scalar Value Types](#scalar-value-types)



<a name="cbnetwork/cloud_adaptive_network.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cbnetwork/cloud_adaptive_network.proto
cloud_adaptive_network.proto


<a name="cbnet.AvailableIPv4PrivateAddressSpaces"></a>

### AvailableIPv4PrivateAddressSpaces



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| recommended_ipv4_private_address_space | [string](#string) |  |  |
| address_space10s | [string](#string) | repeated |  |
| address_space172s | [string](#string) | repeated |  |
| address_space192s | [string](#string) | repeated |  |






<a name="cbnet.CLADNetID"></a>

### CLADNetID
An ID of Cloud Adpative Network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |






<a name="cbnet.CLADNetSpecification"></a>

### CLADNetSpecification
A specification of Cloud Adaptive Network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| ipv4_address_space | [string](#string) |  |  |
| description | [string](#string) |  |  |






<a name="cbnet.CLADNetSpecifications"></a>

### CLADNetSpecifications
A list of Cloud Adaptive Network specification


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cladnet_specifications | [CLADNetSpecification](#cbnet.CLADNetSpecification) | repeated |  |






<a name="cbnet.IPNetworks"></a>

### IPNetworks
A list of IP networks


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip_networks | [string](#string) | repeated |  |





 

 

 


<a name="cbnet.CloudAdaptiveNetwork"></a>

### CloudAdaptiveNetwork

A Cloud Adaptive Network API
//////////////////////////////////////////

The API manages Cloud Adaptive Network (shortly CLADNet).

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| sayHello | [.google.protobuf.Empty](#google.protobuf.Empty) | [.google.protobuf.StringValue](#google.protobuf.StringValue) | Return Say Hello |
| getCLADNet | [CLADNetID](#cbnet.CLADNetID) | [CLADNetSpecification](#cbnet.CLADNetSpecification) | Return a specific CLADNet |
| getCLADNetList | [.google.protobuf.Empty](#google.protobuf.Empty) | [CLADNetSpecifications](#cbnet.CLADNetSpecifications) | Return a specific CLADNet |
| createCLADNet | [CLADNetSpecification](#cbnet.CLADNetSpecification) | [CLADNetSpecification](#cbnet.CLADNetSpecification) | Create a new CLADNet |
| recommendAvailableIPv4PrivateAddressSpaces | [IPNetworks](#cbnet.IPNetworks) | [AvailableIPv4PrivateAddressSpaces](#cbnet.AvailableIPv4PrivateAddressSpaces) | Returns available IPv4 private address spaces |

 



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

