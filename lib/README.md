All the business logic for pggat happens here.

# Folder overview
In general, the top level folder will hold the interface. A subfolder (generally the plural version of the parent folder's name) will hold versioned implementations.

## auth
All authentication functions. Protocol unspecific.

## bouncer
All routing: accepting frontends, backends, and handling transactions.

## middleware
Intercept packets and perform operations on them

## perror
Special postgres error types

## rob
A fair-share scheduler

## util
Project generic helper structures and functions

## zap
Zero allocation packet handling

### zap/packets
Packet reading/writing helpers
