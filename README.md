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

## Documentation ...
