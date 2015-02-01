# The qDHCPv4 server
qDHCPv4 is DHCPv4 server for public places that use unmanaged switches.

## Features
* Dynamically assigned lease time based on schedule (time points when all users must free or extend their lease). Useful for preventing exhaustion of free leases on public places.
* User Classes / Subnets based on `Host Name` (DHCP Option #12).

### < sarcasm /> :-)
* Full High Super+ Speed (as USB). Generate 41 Mbps `offer` / `ACK` flood by incoming 74 Mbps `Discover` / `Request` flood.
* Goes beyond the RFC scope. You can see clear logic in [one](https://github.com/ZiroKyl/qDHCPv4/blob/3252365498792b319557cc86e6f209057465c12f/qDHCPv4.go#L302) of first commits.
* Need only 34 GiB virtual memory to run (on 64bit systems).
* Doesn't require permanent data storage.
* Code-style: `JavaScript <-> Go <-> C`.

## Usage
For Windows 64bit [download](https://github.com/ZiroKyl/qDHCPv4/releases), unpack and run:
```cmd
qDHCPv4.exe -conf=<path to config.json>
```
For other systems: [compile binary](https://golang.org/doc/install) `go get` `go build`.

## Documentation
### LeaseEndTime scheduler logic
![LeaseEndTime](//raw.githubusercontent.com/ZiroKyl/qDHCPv4/dev/doc/scheduler_logic.svg)
### LeaseEndTime 3 user example
![LeaseEndTime](//raw.githubusercontent.com/ZiroKyl/qDHCPv4/dev/doc/example_3user.svg)
### Config file
See [example_config.json](example_config.json).
#### globalOptions
Transmitted options to DHCP client. List of all options and descriptions: [dhcpd options](http://linux.die.net/man/5/dhcp-options), [DHCP and BOOTP parameters](http://www.iana.org/assignments/bootp-dhcp-parameters/bootp-dhcp-parameters.xhtml), [dhcpd options RU](http://www.opennet.ru:8101/man.shtml?topic=dhcp-options&category=5&russian=0), [DHCP options in JSON](https://github.com/ZiroKyl/reflectDHCP).
Example:
```json
"globalOptions":{
	"ServerIdentifier": "194.188.64.28",
	"SubnetMask": "255.255.0.0",
	"Router": ["194.188.64.8"],
	"DomainNameServer": ["194.188.64.8"]
	}
```
#### leaseEndTime
Time when all clients [must renew](#leaseendtime-scheduler-logic) the lease. Format `hh:mm`.
Example:
```json
"leaseEndTime":[
	"11:35",
    "12:50",
    "17:23"
]
```
#### devices
Define user Classes / Subnets based on first two chars from `Host Name` (DHCP Option #12).
Parameters:
- **name** - first two chars from `Host Name` (`android-a70378b9bf61c919` -> `an`; `iPhone-Jon`, `iPad-Jon` -> `iP`; `Windows-Phone` -> `Wi`)
- **startIP** - first client IP in this scope (Class / Subnet)
- **rangeIP** - count of all IP's in this scope (Class / Subnet)
Example:
```json
"devices":[
	{"name": "an", "startIP": "194.188.36.1", "rangeIP": 1024},
    {"name": "iP", "startIP": "194.188.40.1", "rangeIP": 1024},
    {"name": "Wi", "startIP": "194.188.44.1", "rangeIP": 1024}
]
```
#### defaultDevice
`startIP` and `rangeIP` for all other devices.
```json
"defaultDevice":{"startIP": "194.188.32.1", "rangeIP": 1024}
```
