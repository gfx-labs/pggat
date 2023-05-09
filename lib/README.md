All the business logic for pggat happens here.

# Folder overview
In general, the top level folder will hold the interface. A subfolder (generally the plural version of the parent folder's name) will hold versioned implementations.

## auth
All authentication functions. Protocol unspecific.

## backend
Connection handler for pggat -> postgres

## frontend
Connection handler for client -> pggat

## perror
Special postgres error types

## pnet
Zero allocation network handling

### pnet/packet
Packet reading/writing helpers

## rob
A fair-share scheduler

## util
Project generic helper structures and functions
